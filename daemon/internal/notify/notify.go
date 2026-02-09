package notify

import (
	"fmt"
	"os/exec"
	"strings"
)

// Config controls notification behavior.
type Config struct {
	Enabled      bool
	Sound        bool
	ShowPreviews bool
}

// Notifier sends macOS notifications via osascript.
type Notifier struct {
	cfg Config
}

// New creates a new notifier with the given config.
func New(cfg Config) *Notifier {
	return &Notifier{cfg: cfg}
}

// Send displays a macOS notification with the given title and message.
func (n *Notifier) Send(title, message string) error {
	if !n.cfg.Enabled {
		return nil
	}

	if n.cfg.ShowPreviews && len(message) > 200 {
		message = message[:200] + "..."
	}

	script := fmt.Sprintf(`display notification %s with title %s`,
		appleScriptString(message), appleScriptString(title))

	if n.cfg.Sound {
		script += ` sound name "default"`
	}

	return exec.Command("osascript", "-e", script).Run()
}

// SendWithSubtitle displays a notification with title, subtitle, and message.
func (n *Notifier) SendWithSubtitle(title, subtitle, message string) error {
	if !n.cfg.Enabled {
		return nil
	}

	if n.cfg.ShowPreviews && len(message) > 200 {
		message = message[:200] + "..."
	}

	script := fmt.Sprintf(`display notification %s with title %s subtitle %s`,
		appleScriptString(message), appleScriptString(title), appleScriptString(subtitle))

	if n.cfg.Sound {
		script += ` sound name "default"`
	}

	return exec.Command("osascript", "-e", script).Run()
}

// appleScriptString escapes a string for AppleScript.
func appleScriptString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}
