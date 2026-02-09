package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/bender/internal/config"
	"github.com/user/bender/internal/llm"
	"github.com/user/bender/internal/logging"
)

// Ensure handler signatures match json.RawMessage types
// json.RawMessage is []byte but Go requires the exact type match

// Clipboard summarization

type summarizePayload struct {
	Content string `json:"content"`
}

type summarizeResult struct {
	Summary string `json:"summary"`
}

func handleClipboardSummarize(ctx context.Context, payload []byte, router *llm.Router) ([]byte, error) {
	var p summarizePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	if p.Content == "" {
		return nil, fmt.Errorf("empty content")
	}

	logging.Info("summarizing clipboard content (%d chars)", len(p.Content))

	resp, err := router.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a concise summarizer. Summarize the following text in 2-3 sentences. Focus on the key points and main ideas. Return only the summary, nothing else."},
			{Role: "user", Content: p.Content},
		},
		Temperature: 0.3,
		MaxTokens:   256,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	result := summarizeResult{Summary: strings.TrimSpace(resp.Content)}
	logging.Info("clipboard summarized successfully")
	return json.Marshal(result)
}

// File classification

type classifyPayload struct {
	Path string `json:"path"`
}

type classifyResult struct {
	Category    string `json:"category"`
	Destination string `json:"destination"`
	Confidence  string `json:"confidence"`
}

func handleFileClassify(ctx context.Context, payload []byte, router *llm.Router, cfg *config.Config) ([]byte, error) {
	var p classifyPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	if p.Path == "" {
		return nil, fmt.Errorf("empty path")
	}

	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	ext := strings.TrimPrefix(filepath.Ext(p.Path), ".")
	name := filepath.Base(p.Path)
	size := info.Size()

	// Try extension-based classification first
	for _, cat := range cfg.AutoFile.Categories {
		for _, catExt := range cat.Extensions {
			if strings.EqualFold(ext, catExt) {
				logging.Info("classified %s as %s (by extension)", name, cat.Name)
				return json.Marshal(classifyResult{
					Category:    cat.Name,
					Destination: filepath.Join(cat.Path, name),
					Confidence:  "high",
				})
			}
		}
	}

	// Fall back to LLM classification
	if !cfg.AutoFile.UseLLMClassification {
		return json.Marshal(classifyResult{
			Category:    "unknown",
			Destination: filepath.Join(cfg.AutoFile.DestinationRoot, name),
			Confidence:  "none",
		})
	}

	// Read content preview for text-like files
	preview := ""
	textExts := map[string]bool{"txt": true, "md": true, "csv": true, "json": true, "xml": true, "html": true, "log": true, "pdf": true}
	if textExts[strings.ToLower(ext)] && size < 1<<20 {
		data, err := os.ReadFile(p.Path)
		if err == nil {
			preview = string(data)
			if len(preview) > 1000 {
				preview = preview[:1000]
			}
		}
	}

	var catDescs []string
	for _, cat := range cfg.AutoFile.Categories {
		desc := cat.Name
		if cat.Description != "" {
			desc += ": " + cat.Description
		} else if len(cat.Extensions) > 0 {
			desc += " (" + strings.Join(cat.Extensions, ", ") + ")"
		}
		catDescs = append(catDescs, "- "+desc)
	}

	prompt := fmt.Sprintf(`Classify this file into one of the available categories.

File: %s
Extension: %s
Size: %d bytes
Content preview: %s

Available categories:
%s

Return ONLY the category name, nothing else.`, name, ext, size, preview, strings.Join(catDescs, "\n"))

	logging.Info("classifying file %s via LLM", name)

	resp, err := router.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a file classification assistant. Return only the exact category name from the provided list."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   32,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	category := strings.TrimSpace(strings.ToLower(resp.Content))

	// Find the matching category for the destination path
	dest := filepath.Join(cfg.AutoFile.DestinationRoot, name)
	confidence := "medium"
	for _, cat := range cfg.AutoFile.Categories {
		if strings.EqualFold(cat.Name, category) {
			dest = filepath.Join(cat.Path, name)
			confidence = "high"
			break
		}
	}

	logging.Info("classified %s as %s (confidence: %s)", name, category, confidence)

	return json.Marshal(classifyResult{
		Category:    category,
		Destination: dest,
		Confidence:  confidence,
	})
}

// File rename

type renamePayload struct {
	Path string `json:"path"`
}

type renameResult struct {
	OriginalName string `json:"original_name"`
	NewName      string `json:"new_name"`
	Reason       string `json:"reason"`
}

func handleFileRename(ctx context.Context, payload []byte, router *llm.Router, cfg *config.Config) ([]byte, error) {
	var p renamePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	if p.Path == "" {
		return nil, fmt.Errorf("empty path")
	}

	info, err := os.Stat(p.Path)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	originalName := filepath.Base(p.Path)
	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)
	size := info.Size()

	// Determine file type for the prompt
	fileType := "file"
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true, ".bmp": true}
	if imageExts[strings.ToLower(ext)] {
		fileType = "image"
	}

	// Read content preview for text files
	preview := ""
	textExts := map[string]bool{".txt": true, ".md": true, ".csv": true, ".json": true, ".xml": true, ".html": true}
	if textExts[strings.ToLower(ext)] && size < 1<<20 {
		data, err := os.ReadFile(p.Path)
		if err == nil {
			preview = string(data)
			if len(preview) > 1000 {
				preview = preview[:1000]
			}
		}
	}

	prompt := fmt.Sprintf(`Generate a descriptive filename for this %s.
Current name: %s
Size: %d bytes
Content preview: %s

Use %s naming convention.
Keep under %d characters.
Return ONLY the new filename without the extension, nothing else.`,
		fileType, nameWithoutExt, size, preview,
		cfg.Rename.NamingConvention, cfg.Rename.MaxLength)

	logging.Info("generating rename for %s via LLM", originalName)

	resp, err := router.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a file naming assistant. Generate descriptive, clean filenames. Return only the filename without extension."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   64,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	newName := strings.TrimSpace(resp.Content)
	// Remove any quotes the LLM might add
	newName = strings.Trim(newName, "\"'`")

	if cfg.Rename.IncludeDate {
		dateStr := info.ModTime().Format("2006-01-02")
		if cfg.Rename.DatePosition == "suffix" {
			newName = newName + "-" + dateStr
		} else {
			newName = dateStr + "-" + newName
		}
	}

	if cfg.Rename.PreserveExtension && ext != "" {
		newName = newName + ext
	}

	logging.Info("suggested rename: %s -> %s", originalName, newName)

	return json.Marshal(renameResult{
		OriginalName: originalName,
		NewName:      newName,
		Reason:       "LLM-generated descriptive name",
	})
}

// Git commit message generation

type commitPayload struct {
	Diff  string   `json:"diff"`
	Files []string `json:"files"`
}

type commitResult struct {
	Message string `json:"message"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func handleGitCommit(ctx context.Context, payload []byte, router *llm.Router, cfg *config.Config) ([]byte, error) {
	var p commitPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	if p.Diff == "" && len(p.Files) == 0 {
		return nil, fmt.Errorf("no diff or files provided")
	}

	diff := p.Diff
	if len(diff) > 8000 {
		diff = diff[:8000] + "\n... (truncated)"
	}

	filesStr := strings.Join(p.Files, ", ")

	var formatInstructions string
	switch cfg.Git.CommitFormat {
	case "conventional":
		formatInstructions = fmt.Sprintf(`Use conventional commit format: type(scope): description
Types: feat, fix, docs, style, refactor, test, chore
Keep subject line under %d characters.`, cfg.Git.MaxSubjectLength)
		if cfg.Git.IncludeBody {
			formatInstructions += fmt.Sprintf(`
Add a body explaining WHY if the change is non-trivial.
Wrap body at %d characters.`, cfg.Git.MaxBodyWidth)
		}
	case "detailed":
		formatInstructions = "Write a detailed commit message with a subject line and bullet points describing each change."
	default:
		formatInstructions = fmt.Sprintf("Write a concise commit message under %d characters.", cfg.Git.MaxSubjectLength)
	}

	prompt := fmt.Sprintf(`Generate a git commit message for the following changes:

Files changed: %s

Diff:
%s

%s`, filesStr, diff, formatInstructions)

	logging.Info("generating commit message for %d files via LLM", len(p.Files))

	resp, err := router.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a git commit message generator. Write clear, accurate commit messages based on the diff provided. Return only the commit message, nothing else."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   256,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	message := strings.TrimSpace(resp.Content)

	// Split into subject and body
	subject := message
	body := ""
	if parts := strings.SplitN(message, "\n\n", 2); len(parts) == 2 {
		subject = parts[0]
		body = parts[1]
	} else if parts := strings.SplitN(message, "\n", 2); len(parts) == 2 {
		subject = parts[0]
		body = strings.TrimSpace(parts[1])
	}

	logging.Info("generated commit message: %s", subject)

	return json.Marshal(commitResult{
		Message: message,
		Subject: subject,
		Body:    body,
	})
}

// Screenshot tagging

type screenshotPayload struct {
	Path string `json:"path"`
}

type screenshotResult struct {
	App           string   `json:"app"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	SuggestedName string   `json:"suggested_name"`
}

func handleScreenshotTag(ctx context.Context, payload []byte, router *llm.Router, cfg *config.Config) ([]byte, error) {
	var p screenshotPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	if p.Path == "" {
		return nil, fmt.Errorf("empty path")
	}

	imgData, err := os.ReadFile(p.Path)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(p.Path))
	mimeType := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	}

	logging.Info("tagging screenshot %s via vision LLM", filepath.Base(p.Path))

	resp, err := router.CompleteWithVision(ctx, llm.VisionRequest{
		CompletionRequest: llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: "system", Content: "You are a screenshot analysis assistant."},
				{Role: "user", Content: `Analyze this screenshot and provide:
1. App or website shown (if identifiable)
2. Brief description of content (under 10 words)
3. Up to 5 relevant tags

Return as JSON: {"app": "", "description": "", "tags": []}`},
			},
			Temperature: 0.2,
			MaxTokens:   256,
		},
		Images: []llm.Image{
			{Data: imgData, MimeType: mimeType},
		},
	}, cfg.Screenshots.VisionProvider)
	if err != nil {
		return nil, fmt.Errorf("vision completion: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	// Try to extract JSON from the response
	if idx := strings.Index(content, "{"); idx >= 0 {
		if end := strings.LastIndex(content, "}"); end > idx {
			content = content[idx : end+1]
		}
	}

	var result screenshotResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If JSON parsing fails, use the raw response
		result = screenshotResult{
			App:         "unknown",
			Description: content,
			Tags:        []string{},
		}
	}

	// Generate suggested filename
	nameParts := []string{}
	if result.App != "" && result.App != "unknown" {
		nameParts = append(nameParts, sanitizeFilename(result.App))
	}
	if result.Description != "" {
		nameParts = append(nameParts, sanitizeFilename(result.Description))
	}
	if len(nameParts) > 0 {
		result.SuggestedName = strings.Join(nameParts, "-") + ext
	} else {
		result.SuggestedName = filepath.Base(p.Path)
	}

	logging.Info("tagged screenshot: app=%s, desc=%s, tags=%v", result.App, result.Description, result.Tags)

	return json.Marshal(result)
}

func sanitizeFilename(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, s)
	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
