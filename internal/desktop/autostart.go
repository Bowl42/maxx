package desktop

import (
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
)

const appName = "maxx"

func getAutoStartApp() *autostart.App {
	exePath, err := os.Executable()
	if err != nil {
		return nil
	}

	return &autostart.App{
		Name:        appName,
		DisplayName: "Maxx Next",
		Exec:        []string{exePath, "--minimized"},
	}
}

func setAutoStart(enable bool) error {
	app := getAutoStartApp()
	if app == nil {
		return nil
	}

	if enable {
		return app.Enable()
	}
	return app.Disable()
}

func isAutoStartEnabled() bool {
	app := getAutoStartApp()
	if app == nil {
		return false
	}
	return app.IsEnabled()
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
