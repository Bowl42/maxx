package converter

import "sync"

// GlobalSettings holds converter-related global configuration.
type GlobalSettings struct {
	CodexInstructionsEnabled bool
}

var (
	globalSettingsMu   sync.RWMutex
	settingsGetterFunc func() (*GlobalSettings, error)
)

// SetGlobalSettingsGetter sets the function to retrieve global settings.
func SetGlobalSettingsGetter(getter func() (*GlobalSettings, error)) {
	globalSettingsMu.Lock()
	defer globalSettingsMu.Unlock()
	settingsGetterFunc = getter
}

// GetGlobalSettings retrieves the current global settings.
func GetGlobalSettings() *GlobalSettings {
	globalSettingsMu.RLock()
	defer globalSettingsMu.RUnlock()
	if settingsGetterFunc == nil {
		return nil
	}
	settings, err := settingsGetterFunc()
	if err != nil {
		return nil
	}
	return settings
}
