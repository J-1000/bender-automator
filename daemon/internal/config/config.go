package config

import (
	"os"
	"path/filepath"

	"github.com/user/bender/internal/keychain"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM           LLMConfig           `yaml:"llm"`
	Clipboard     ClipboardConfig     `yaml:"clipboard"`
	AutoFile      AutoFileConfig      `yaml:"auto_file"`
	Rename        RenameConfig        `yaml:"rename"`
	Git           GitConfig           `yaml:"git"`
	Screenshots   ScreenshotsConfig   `yaml:"screenshots"`
	Queue         QueueConfig         `yaml:"queue"`
	Logging       LoggingConfig       `yaml:"logging"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type LLMConfig struct {
	DefaultProvider string                     `yaml:"default_provider"`
	Providers       map[string]*ProviderConfig `yaml:"providers"`
}

type ProviderConfig struct {
	Enabled        bool   `yaml:"enabled"`
	BaseURL        string `yaml:"base_url"`
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	VisionModel    string `yaml:"vision_model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

type ClipboardConfig struct {
	Enabled           bool `yaml:"enabled"`
	MinLength         int  `yaml:"min_length"`
	DebounceMs        int  `yaml:"debounce_ms"`
	AutoSummarize     bool `yaml:"auto_summarize"`
	Notification      bool `yaml:"notification"`
	NotificationSound bool `yaml:"notification_sound"`
}

type AutoFileConfig struct {
	Enabled              bool       `yaml:"enabled"`
	WatchDirs            []string   `yaml:"watch_dirs"`
	DestinationRoot      string     `yaml:"destination_root"`
	IgnoreHidden         bool       `yaml:"ignore_hidden"`
	ExcludePatterns      []string   `yaml:"exclude_patterns"`
	Categories           []Category `yaml:"categories"`
	UseLLMClassification bool       `yaml:"use_llm_classification"`
	AutoMove             bool       `yaml:"auto_move"`
	AutoRename           bool       `yaml:"auto_rename"`
	SettleDelayMs        int        `yaml:"settle_delay_ms"`
}

type Category struct {
	Name        string   `yaml:"name"`
	Path        string   `yaml:"path"`
	Extensions  []string `yaml:"extensions"`
	Description string   `yaml:"description"`
}

type RenameConfig struct {
	NamingConvention  string `yaml:"naming_convention"`
	IncludeDate       bool   `yaml:"include_date"`
	DateFormat        string `yaml:"date_format"`
	DatePosition      string `yaml:"date_position"`
	MaxLength         int    `yaml:"max_length"`
	PreserveExtension bool   `yaml:"preserve_extension"`
}

type GitConfig struct {
	Enabled           bool   `yaml:"enabled"`
	AutoInstallHooks  bool   `yaml:"auto_install_hooks"`
	CommitFormat      string `yaml:"commit_format"`
	IncludeScope      bool   `yaml:"include_scope"`
	IncludeBody       bool   `yaml:"include_body"`
	MaxSubjectLength  int    `yaml:"max_subject_length"`
	MaxBodyWidth      int    `yaml:"max_body_width"`
	IncludeDiffInBody bool   `yaml:"include_diff_in_body"`
}

type ScreenshotsConfig struct {
	Enabled         bool   `yaml:"enabled"`
	WatchDir        string `yaml:"watch_dir"`
	Destination     string `yaml:"destination"`
	Rename          bool   `yaml:"rename"`
	AddMetadataTags bool   `yaml:"add_metadata_tags"`
	UseVision       bool   `yaml:"use_vision"`
	VisionProvider  string `yaml:"vision_provider"`
	SettleDelayMs   int    `yaml:"settle_delay_ms"`
}

type QueueConfig struct {
	MaxConcurrent         int `yaml:"max_concurrent"`
	DefaultTimeoutSeconds int `yaml:"default_timeout_seconds"`
	MaxRetries            int `yaml:"max_retries"`
	RetryDelaySeconds     int `yaml:"retry_delay_seconds"`
}

type LoggingConfig struct {
	Level             string `yaml:"level"`
	MaxSizeMB         int    `yaml:"max_size_mb"`
	MaxFiles          int    `yaml:"max_files"`
	IncludeTimestamps bool   `yaml:"include_timestamps"`
}

type NotificationsConfig struct {
	Enabled      bool `yaml:"enabled"`
	Sound        bool `yaml:"sound"`
	ShowPreviews bool `yaml:"show_previews"`
}

func Load(path string) (*Config, error) {
	path = expandPath(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.setDefaults()
	cfg.expandPaths()
	cfg.resolveSecrets()

	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.LLM.DefaultProvider == "" {
		c.LLM.DefaultProvider = "ollama"
	}
	if c.Clipboard.MinLength == 0 {
		c.Clipboard.MinLength = 500
	}
	if c.Clipboard.DebounceMs == 0 {
		c.Clipboard.DebounceMs = 1000
	}
	if c.Queue.MaxConcurrent == 0 {
		c.Queue.MaxConcurrent = 2
	}
	if c.Queue.DefaultTimeoutSeconds == 0 {
		c.Queue.DefaultTimeoutSeconds = 30
	}
	if c.Queue.MaxRetries == 0 {
		c.Queue.MaxRetries = 3
	}
	if c.AutoFile.SettleDelayMs == 0 {
		c.AutoFile.SettleDelayMs = 3000
	}
	if c.Screenshots.SettleDelayMs == 0 {
		c.Screenshots.SettleDelayMs = 2000
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.MaxSizeMB == 0 {
		c.Logging.MaxSizeMB = 10
	}
	if c.Logging.MaxFiles == 0 {
		c.Logging.MaxFiles = 5
	}
}

func (c *Config) expandPaths() {
	for i := range c.AutoFile.WatchDirs {
		c.AutoFile.WatchDirs[i] = expandPath(c.AutoFile.WatchDirs[i])
	}
	c.AutoFile.DestinationRoot = expandPath(c.AutoFile.DestinationRoot)
	for i := range c.AutoFile.Categories {
		c.AutoFile.Categories[i].Path = expandPath(c.AutoFile.Categories[i].Path)
	}
	c.Screenshots.WatchDir = expandPath(c.Screenshots.WatchDir)
	c.Screenshots.Destination = expandPath(c.Screenshots.Destination)
}

func (c *Config) resolveSecrets() {
	for name, prov := range c.LLM.Providers {
		if prov.APIKey != "" {
			resolved, err := keychain.Resolve(prov.APIKey)
			if err == nil {
				c.LLM.Providers[name].APIKey = resolved
			}
			// If resolve fails for a keychain: ref, leave the original value
			// so the error surfaces when the provider tries to use it
		}
	}
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
