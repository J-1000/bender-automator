package keychain

import (
	"fmt"
	"os/exec"
	"strings"
)

const serviceName = "bender"

// Get retrieves a password from the macOS Keychain for the given account.
func Get(account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", serviceName,
		"-a", account,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain: key not found for %q", account)
	}
	return strings.TrimSpace(string(out)), nil
}

// Set stores a password in the macOS Keychain for the given account.
// If the entry already exists, it is updated.
func Set(account, password string) error {
	// Try to delete existing entry first (ignore errors if not found)
	del := exec.Command("security", "delete-generic-password",
		"-s", serviceName,
		"-a", account,
	)
	del.Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", serviceName,
		"-a", account,
		"-w", password,
		"-U",
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain: failed to store key for %q: %w", account, err)
	}
	return nil
}

// Delete removes a password from the macOS Keychain for the given account.
func Delete(account string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", serviceName,
		"-a", account,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keychain: key not found for %q", account)
	}
	return nil
}

// Resolve checks if a value is a keychain reference (prefixed with "keychain:")
// and resolves it. If it's a plain value, it's returned as-is.
func Resolve(value string) (string, error) {
	if !strings.HasPrefix(value, "keychain:") {
		return value, nil
	}
	account := strings.TrimPrefix(value, "keychain:")
	return Get(account)
}
