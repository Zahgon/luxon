package intl

import (
	"encoding/json"
	"sync"
)

type throwData struct {
	Throw []string `json:"throw"`
}

func (t throwData) err() error {
	if len(t.Throw) == 2 {
		return &Error{Name: t.Throw[0], Message: t.Throw[1]}
	}
	return nil
}

type resolvedEntry struct {
	DTF       json.RawMessage `json:"dtf"`
	NF        json.RawMessage `json:"nf"`
	Week      json.RawMessage `json:"week"`
	LocaleObj json.RawMessage `json:"localeObj"`
}

type numSymbols struct {
	NS          string   `json:"ns"`
	Digits      []string `json:"digits"`
	Minus       string   `json:"minus"`
	Plus        string   `json:"plus"`
	Decimal     string   `json:"decimal"`
	Group       string   `json:"group"`
	MinGrouping int      `json:"minGrouping"`
	Secondary   int      `json:"secondary"`
	NaN         string   `json:"nan"`
	Inf         string   `json:"inf"`
}

type relPatterns struct {
	Future map[string]string `json:"future"`
	Past   map[string]string `json:"past"`
	Auto   map[string]string `json:"auto"`
}

type localeData struct {
	Resolved map[string]resolvedEntry                                     `json:"resolved"`
	Num      map[string]*numSymbols                                       `json:"num"`
	Units    map[string]map[string]map[string]string                      `json:"units"`
	Rel      map[string]map[string]*relPatterns                           `json:"rel"`
	Lists    map[string]map[string]map[string]string                      `json:"lists"`
	Plural   map[string]string                                            `json:"plural"`
	Errors   map[string]throwData                                         `json:"errors"`
	System   struct{ Calendar, Locale, NumberingSystem, TimeZone string } `json:"system"`
}

var (
	locOnce sync.Once
	locData localeData
)

func locales() *localeData {
	locOnce.Do(func() { loadJSON("locale", &locData) })
	return &locData
}

// Unsupported reports a locale string or option combination the recorded ICU data does not cover.
type Unsupported struct{ What string }

func (u *Unsupported) Error() string { return "intl: no ICU data for " + u.What }

func knownError(name string) error {
	if t, ok := locales().Errors[name]; ok {
		return t.err()
	}
	return rangeError(name)
}

// IncorrectLocale is the RangeError ICU raises for a malformed locale tag.
func IncorrectLocale() error { return knownError("locale") }

// SystemLocale is the default locale of the runtime the original was recorded on.
func SystemLocale() string { return locales().System.Locale }

// SystemTimeZone is the default time zone of the runtime the original was recorded on.
func SystemTimeZone() string { return locales().System.TimeZone }

// ResolvedDTF is what new Intl.DateTimeFormat(locale).resolvedOptions() reports.
type ResolvedDTF struct {
	Locale          string `json:"locale"`
	Calendar        string `json:"calendar"`
	NumberingSystem string `json:"numberingSystem"`
	TimeZone        string `json:"timeZone"`
}

func entry(locale string) (*resolvedEntry, error) {
	e, ok := locales().Resolved[locale]
	if !ok {
		return nil, &Unsupported{What: "locale " + locale}
	}
	return &e, nil
}

func decodeOrThrow(raw json.RawMessage, v any) error {
	var t throwData
	if json.Unmarshal(raw, &t) == nil && len(t.Throw) == 2 {
		return t.err()
	}
	return json.Unmarshal(raw, v)
}

// ResolveDateTimeFormat resolves a locale string the way Intl.DateTimeFormat does.
func ResolveDateTimeFormat(locale string) (ResolvedDTF, error) {
	if locale == "" {
		s := locales().System
		return ResolvedDTF{Locale: s.Locale, Calendar: s.Calendar, NumberingSystem: s.NumberingSystem, TimeZone: s.TimeZone}, nil
	}
	e, err := entry(locale)
	if err != nil {
		return ResolvedDTF{}, err
	}
	var r ResolvedDTF
	if err := decodeOrThrow(e.DTF, &r); err != nil {
		return ResolvedDTF{}, err
	}
	return r, nil
}

// ResolvedNF is what new Intl.NumberFormat(locale).resolvedOptions() reports.
type ResolvedNF struct {
	Locale          string `json:"locale"`
	NumberingSystem string `json:"numberingSystem"`
}

// ResolveNumberFormat resolves a locale string the way Intl.NumberFormat does.
func ResolveNumberFormat(locale string) (ResolvedNF, error) {
	if locale == "" {
		s := locales().System
		return ResolvedNF{Locale: s.Locale, NumberingSystem: s.NumberingSystem}, nil
	}
	e, err := entry(locale)
	if err != nil {
		return ResolvedNF{}, err
	}
	var r ResolvedNF
	if err := decodeOrThrow(e.NF, &r); err != nil {
		return ResolvedNF{}, err
	}
	return r, nil
}

// WeekInfo is Intl.Locale.prototype.getWeekInfo(); MinimalDays is nil when ICU omits it.
type WeekInfo struct {
	FirstDay    int   `json:"firstDay"`
	MinimalDays *int  `json:"minimalDays"`
	Weekend     []int `json:"weekend"`
}

// LocaleWeekInfo returns the week data of new Intl.Locale(locale).
func LocaleWeekInfo(locale string) (WeekInfo, error) {
	e, err := entry(locale)
	if err != nil {
		return WeekInfo{}, err
	}
	var w WeekInfo
	if err := decodeOrThrow(e.Week, &w); err != nil {
		return WeekInfo{}, err
	}
	return w, nil
}

func numberSymbols(locale string) (*numSymbols, error) {
	if locale == "" {
		locale = SystemLocale()
	}
	if _, err := ResolveNumberFormat(locale); err != nil {
		return nil, err
	}
	n, ok := locales().Num[locale]
	if !ok {
		return nil, &Unsupported{What: "number symbols for " + locale}
	}
	return n, nil
}
