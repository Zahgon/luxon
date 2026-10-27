package luxon

// settings.js

import (
	"math"
	"time"
)

var (
	settingsNow                        = func() float64 { return float64(time.Now().UnixMilli()) }
	settingsDefaultZone            any = "system"
	settingsDefaultLocale          any = nil
	settingsDefaultNumberingSystem any = nil
	settingsDefaultOutputCalendar  any = nil
	settingsTwoDigitCutoffYear         = 60.0
	settingsThrowOnInvalid         any = Undefined
	settingsDefaultWeekSettings    *weekSettings
)

// SettingsType holds Luxon's global settings; use the Settings value.
type SettingsType struct{}

// Settings is Luxon's Settings class. Getters are methods named after the JavaScript property
// (Settings.DefaultZone()); setters are the same name prefixed with Set.
var Settings SettingsType

// Now is the Settings.now function, which returns the current epoch milliseconds.
func (SettingsType) Now() func() float64 { return settingsNow }

// SetNow is Settings.now = n.
func (SettingsType) SetNow(n func() float64) { settingsNow = n }

// DefaultZone is Settings.defaultZone, normalized.
func (SettingsType) DefaultZone() Zone { return normalizeZone(settingsDefaultZone, systemInstance) }

// SetDefaultZone is Settings.defaultZone = zone (a Zone, a string, a number or nil).
func (SettingsType) SetDefaultZone(zone any) { settingsDefaultZone = zone }

// DefaultLocale is Settings.defaultLocale.
func (SettingsType) DefaultLocale() any { return settingsDefaultLocale }

// SetDefaultLocale is Settings.defaultLocale = locale.
func (SettingsType) SetDefaultLocale(locale any) { settingsDefaultLocale = locale }

// DefaultNumberingSystem is Settings.defaultNumberingSystem.
func (SettingsType) DefaultNumberingSystem() any { return settingsDefaultNumberingSystem }

// SetDefaultNumberingSystem is Settings.defaultNumberingSystem = numberingSystem.
func (SettingsType) SetDefaultNumberingSystem(numberingSystem any) {
	settingsDefaultNumberingSystem = numberingSystem
}

// DefaultOutputCalendar is Settings.defaultOutputCalendar.
func (SettingsType) DefaultOutputCalendar() any { return settingsDefaultOutputCalendar }

// SetDefaultOutputCalendar is Settings.defaultOutputCalendar = outputCalendar.
func (SettingsType) SetDefaultOutputCalendar(outputCalendar any) {
	settingsDefaultOutputCalendar = outputCalendar
}

// DefaultWeekSettings is Settings.defaultWeekSettings: nil, or an Obj {firstDay, minimalDays, weekend}.
func (SettingsType) DefaultWeekSettings() any {
	if settingsDefaultWeekSettings == nil {
		return nil
	}
	return weekSettingsObj(settingsDefaultWeekSettings)
}

// SetDefaultWeekSettings is Settings.defaultWeekSettings = weekSettings.
func (SettingsType) SetDefaultWeekSettings(weekSettings any) {
	settingsDefaultWeekSettings = validateWeekSettings(weekSettings)
}

// TwoDigitCutoffYear is Settings.twoDigitCutoffYear.
func (SettingsType) TwoDigitCutoffYear() float64 { return settingsTwoDigitCutoffYear }

// SetTwoDigitCutoffYear is Settings.twoDigitCutoffYear = cutoffYear.
func (SettingsType) SetTwoDigitCutoffYear(cutoffYear any) {
	settingsTwoDigitCutoffYear = math.Mod(toNumber(cutoffYear), 100)
}

// ThrowOnInvalid is Settings.throwOnInvalid.
func (SettingsType) ThrowOnInvalid() any { return settingsThrowOnInvalid }

// SetThrowOnInvalid is Settings.throwOnInvalid = t.
func (SettingsType) SetThrowOnInvalid(t any) { settingsThrowOnInvalid = t }

// ResetCaches is Settings.resetCaches().
func (SettingsType) ResetCaches() {
	localeResetCache()
	IANAZoneResetCache()
	DateTimeResetCache()
	resetDigitRegexCache()
}

func weekSettingsObj(ws *weekSettings) Obj {
	weekend := make([]any, len(ws.weekend))
	for i, d := range ws.weekend {
		weekend[i] = d
	}
	return Obj{"firstDay", ws.firstDay, "minimalDays", ws.minimalDays, "weekend", weekend}
}
