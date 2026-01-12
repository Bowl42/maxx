package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	autoStartKey        = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName             = "maxx-next"
	registryPathFormat  = `"%s" --minimized`
)

func setAutoStart(enable bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		value := fmt.Sprintf(registryPathFormat, exePath)
		if err := key.SetStringValue(appName, value); err != nil {
			return fmt.Errorf("failed to set registry value: %w", err)
		}

		return nil
	}

	if err := key.DeleteValue(appName); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to delete registry value: %w", err)
	}

	return nil
}

func isAutoStartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.READ)
	if err != nil {
		return false
	}
	defer key.Close()

	value, _, err := key.GetStringValue(appName)
	if err != nil {
		return false
	}

	if value == "" {
		return false
	}

	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	expectedValue := fmt.Sprintf(registryPathFormat, exePath)

	normalizedActual := strings.ReplaceAll(value, "\\", "/")
	_ = strings.ReplaceAll(expectedValue, "\\", "/") // normalizedExpected unused

	return strings.Contains(normalizedActual, filepath.Base(exePath))
}

func getExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", err
	}

	exePath = filepath.Clean(exePath)

	return exePath, nil
}
