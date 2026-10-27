package luxon

// impl/locale.js: Luxon's bridge to Intl, here backed by internal/intl.

import (
	"math"
	"strings"

	"github.com/moment/luxon-go/internal/intl"
)

// locale is impl/locale.js's Locale.
type locale struct {
	locale          string
	numberingSystem *string
	outputCalendar  *string
	weekSettings    *weekSettings
	intl            string
	specifiedLocale any
	fastNumbersSet  bool
	fastNumbersVal  bool
}

func localeResetCache() {}

// optString turns a JavaScript option value into a string pointer: nil for any falsy value.
func optString(v any) *string {
	if !truthy(v) {
		return nil
	}
	s := toJSString(v)
	return &s
}

func orStr(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}

// newLocaleFromOpts is Locale.fromOpts(opts).
func newLocaleFromOpts(opts *jsObj) *locale {
	return createLocale(opts.get("locale"), opts.get("numberingSystem"), opts.get("outputCalendar"), opts.get("weekSettings"), truthy(opts.get("defaultToEN")))
}

// newLocaleFromObject is Locale.fromObject({locale, numberingSystem, outputCalendar, weekSettings}).
func newLocaleFromObject(opts *jsObj) *locale {
	if opts == nil {
		opts = newObj()
	}
	return createLocale(opts.get("locale"), opts.get("numberingSystem"), opts.get("outputCalendar"), opts.get("weekSettings"), false)
}

// createLocale is Locale.create.
func createLocale(loc, numberingSystem, outputCalendar, weekSettingsArg any, defaultToEN bool) *locale {
	specifiedLocale := loc
	if !truthy(specifiedLocale) {
		specifiedLocale = settingsDefaultLocale
	}
	localeR := specifiedLocale
	if !truthy(localeR) {
		if defaultToEN {
			localeR = "en-US"
		} else {
			localeR = intl.SystemLocale()
		}
	}
	nsR := numberingSystem
	if !truthy(nsR) {
		nsR = settingsDefaultNumberingSystem
	}
	calR := outputCalendar
	if !truthy(calR) {
		calR = settingsDefaultOutputCalendar
	}
	ws := validateWeekSettings(weekSettingsArg)
	if ws == nil {
		ws = settingsDefaultWeekSettings
	}
	return newLocale(localeR, optString(nsR), optString(calR), ws, specifiedLocale)
}

func newLocale(loc any, numbering, outputCalendar *string, ws *weekSettings, specifiedLocale any) *locale {
	s, ok := loc.(string)
	if !ok {
		throw("TypeError", "localeStr.indexOf is not a function")
	}
	parsed, parsedNS, parsedCal := parseLocaleString(s)
	l := &locale{locale: parsed, weekSettings: ws, specifiedLocale: specifiedLocale}
	l.numberingSystem = orStr(numbering, parsedNS)
	l.outputCalendar = orStr(outputCalendar, parsedCal)
	l.intl = intlConfigString(l.locale, l.numberingSystem, l.outputCalendar)
	return l
}

// resolveDTF is new Intl.DateTimeFormat(loc).resolvedOptions().
func resolveDTF(loc string) intl.ResolvedDTF {
	r, err := intl.ResolveDateTimeFormat(loc)
	raiseIntl(err)
	return r
}

// parseLocaleString splits off a -u- extension, resolving it the way ICU does.
func parseLocaleString(localeStr string) (string, *string, *string) {
	if x := strings.Index(localeStr, "-x-"); x != -1 {
		localeStr = localeStr[:x]
	}
	u := strings.Index(localeStr, "-u-")
	if u == -1 {
		return localeStr, nil, nil
	}
	selected := localeStr
	r, err := intl.ResolveDateTimeFormat(localeStr)
	if _, isJS := err.(*intl.Error); isJS {
		selected = localeStr[:u]
		r, err = intl.ResolveDateTimeFormat(selected)
	}
	raiseIntl(err)
	return selected, strp(r.NumberingSystem), strp(r.Calendar)
}

func intlConfigString(localeStr string, numberingSystem, outputCalendar *string) string {
	if outputCalendar == nil && numberingSystem == nil {
		return localeStr
	}
	if !strings.Contains(localeStr, "-u-") {
		localeStr += "-u"
	}
	if outputCalendar != nil {
		localeStr += "-ca-" + *outputCalendar
	}
	if numberingSystem != nil {
		localeStr += "-nu-" + *numberingSystem
	}
	return localeStr
}

func (l *locale) fastNumbers() bool {
	if !l.fastNumbersSet {
		l.fastNumbersVal = l.supportsFastNumbers()
		l.fastNumbersSet = true
	}
	return l.fastNumbersVal
}

func (l *locale) supportsFastNumbers() bool {
	if l.numberingSystem != nil && *l.numberingSystem != "latn" {
		return false
	}
	return (l.numberingSystem != nil && *l.numberingSystem == "latn") || l.locale == "" ||
		strings.HasPrefix(l.locale, "en") || resolveDTF(l.locale).NumberingSystem == "latn"
}

func (l *locale) listingMode() string {
	noWeirdness := (l.numberingSystem == nil || *l.numberingSystem == "latn") &&
		(l.outputCalendar == nil || *l.outputCalendar == "gregory")
	if l.isEnglish() && noWeirdness {
		return "en"
	}
	return "intl"
}

// clone is Locale.clone(alts): alts is nil (undefined), a *jsObj, or another *locale.
func (l *locale) clone(alts any) *locale {
	var get func(string) any
	switch a := alts.(type) {
	case nil, undefinedType:
		return l
	case *jsObj:
		if a == nil || len(a.keys) == 0 {
			return l
		}
		get = a.get
	case *locale:
		get = func(k string) any {
			switch k {
			case "locale":
				return a.locale
			case "numberingSystem":
				if a.numberingSystem == nil {
					return nil
				}
				return *a.numberingSystem
			case "outputCalendar":
				if a.outputCalendar == nil {
					return nil
				}
				return *a.outputCalendar
			case "weekSettings":
				if a.weekSettings == nil {
					return nil
				}
				return a.weekSettings
			}
			return Undefined
		}
	default:
		return l
	}
	loc := get("locale")
	if !truthy(loc) {
		loc = l.specifiedLocale
	}
	ns := get("numberingSystem")
	if !truthy(ns) {
		ns = ptrAny(l.numberingSystem)
	}
	cal := get("outputCalendar")
	if !truthy(cal) {
		cal = ptrAny(l.outputCalendar)
	}
	ws := validateWeekSettings(get("weekSettings"))
	if ws == nil {
		ws = l.weekSettings
	}
	return createLocale(loc, ns, cal, ws, truthy(get("defaultToEN")))
}

func ptrAny(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func (l *locale) redefaultToEN(alts *jsObj) *locale {
	o := newObj()
	o.assign(alts)
	o.set("defaultToEN", true)
	return l.clone(o)
}

func (l *locale) redefaultToSystem(alts *jsObj) *locale {
	o := newObj()
	o.assign(alts)
	o.set("defaultToEN", false)
	return l.clone(o)
}

func mapMonths(f func(*DateTime) any) []any {
	out := make([]any, 0, 12)
	for i := 1; i <= 12; i++ {
		out = append(out, f(DateTimeUtc(2009, i, 1)))
	}
	return out
}

func mapWeekdays(f func(*DateTime) any) []any {
	out := make([]any, 0, 7)
	for i := 1; i <= 7; i++ {
		out = append(out, f(DateTimeUtc(2016, 11, 13+i)))
	}
	return out
}

func stringsToAny(s []string) []any {
	if s == nil {
		return nil
	}
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// months is Locale.months(length, format); nil is null. Elements are strings, or nil where ICU
// shows no month part.
func (l *locale) months(length any, format bool) []any {
	if l.listingMode() == "en" {
		return stringsToAny(englishMonths(length))
	}
	special := l.intl == "ja" || strings.HasPrefix(l.intl, "ja-")
	format = format && !special
	opts := Obj{"month", length}
	if format {
		opts = Obj{"month", length, "day", "numeric"}
	}
	if special {
		return mapMonths(func(dt *DateTime) any { return l.dtFormatter(dt, objFrom(opts)).format() })
	}
	return mapMonths(func(dt *DateTime) any { return strOrNil(l.extract(dt, objFrom(opts), "month")) })
}

func (l *locale) weekdays(length any, format bool) []any {
	if l.listingMode() == "en" {
		return stringsToAny(englishWeekdays(length))
	}
	opts := Obj{"weekday", length}
	if format {
		opts = Obj{"weekday", length, "year", "numeric", "month", "long", "day", "numeric"}
	}
	return mapWeekdays(func(dt *DateTime) any { return strOrNil(l.extract(dt, objFrom(opts), "weekday")) })
}

func (l *locale) meridiems() []any {
	if l.listingMode() == "en" {
		return stringsToAny(meridiemsEn)
	}
	opts := Obj{"hour", "numeric", "hourCycle", "h12"}
	out := []any{}
	for _, dt := range []*DateTime{DateTimeUtc(2016, 11, 13, 9), DateTimeUtc(2016, 11, 13, 19)} {
		out = append(out, strOrNil(l.extract(dt, objFrom(opts), "dayperiod")))
	}
	return out
}

func (l *locale) eras(length any) []any {
	if l.listingMode() == "en" {
		return stringsToAny(englishEras(length))
	}
	opts := Obj{"era", length}
	out := []any{}
	for _, dt := range []*DateTime{DateTimeUtc(-40, 1, 1), DateTimeUtc(2017, 1, 1)} {
		out = append(out, strOrNil(l.extract(dt, objFrom(opts), "era")))
	}
	return out
}

func strOrNil(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// extract is the value of the first part of the given (lowercased) type, or nil.
func (l *locale) extract(dt *DateTime, intlOpts *jsObj, field string) *string {
	for _, p := range l.dtFormatter(dt, intlOpts).formatToParts() {
		if jsToLower(p.Type) == field {
			v := p.Value
			return &v
		}
	}
	return nil
}

func (l *locale) numberFormatter(opts *jsObj) *polyNumberFormatter {
	return newPolyNumberFormatter(l.intl, truthy(opts.get("forceSimple")) || l.fastNumbers(), opts)
}

func (l *locale) dtFormatter(dt *DateTime, intlOpts *jsObj) *polyDateFormatter {
	if intlOpts == nil {
		intlOpts = newObj()
	}
	return newPolyDateFormatter(dt, l.intl, intlOpts)
}

func (l *locale) relFormatter(opts *jsObj) *polyRelFormatter {
	return newPolyRelFormatter(l.intl, l.isEnglish(), opts)
}

func (l *locale) listFormatter(opts *jsObj) *intl.ListFormat {
	typ, style := "", ""
	if v := opts.get("type"); !isUndef(v) {
		typ = toJSString(v)
	}
	if v := opts.get("style"); !isUndef(v) {
		style = toJSString(v)
	}
	lf, err := intl.NewListFormat(l.intl, typ, style)
	raiseIntl(err)
	return lf
}

func (l *locale) isEnglish() bool {
	return l.locale == "en" || jsToLower(l.locale) == "en-us" || strings.HasPrefix(resolveDTF(l.intl).Locale, "en-us")
}

var fallbackWeekSettings = &weekSettings{firstDay: 1, minimalDays: 4, weekend: []float64{6, 7}}

func (l *locale) getWeekSettings() *weekSettings {
	if l.weekSettings != nil {
		return l.weekSettings
	}
	w, err := intl.LocaleWeekInfo(l.locale)
	raiseIntl(err)
	ws := &weekSettings{firstDay: float64(w.FirstDay), minimalDays: 4}
	if w.MinimalDays != nil {
		ws.minimalDays = float64(*w.MinimalDays)
	}
	for _, d := range w.Weekend {
		ws.weekend = append(ws.weekend, float64(d))
	}
	return ws
}

func (l *locale) getStartOfWeek() float64        { return l.getWeekSettings().firstDay }
func (l *locale) getMinDaysInFirstWeek() float64 { return l.getWeekSettings().minimalDays }
func (l *locale) getWeekendDays() []float64      { return l.getWeekSettings().weekend }

func eqStrPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (l *locale) equals(o *locale) bool {
	return l.locale == o.locale && eqStrPtr(l.numberingSystem, o.numberingSystem) && eqStrPtr(l.outputCalendar, o.outputCalendar)
}

func (l *locale) String() string {
	ns, cal := "null", "null"
	if l.numberingSystem != nil {
		ns = *l.numberingSystem
	}
	if l.outputCalendar != nil {
		cal = *l.outputCalendar
	}
	return "Locale(" + l.locale + ", " + ns + ", " + cal + ")"
}

// ---------------------------------------------------------------------------------------------
// PolyNumberFormatter

type polyNumberFormatter struct {
	padTo float64
	floor bool
	inf   *intl.NumberFormat
}

func newPolyNumberFormatter(intlLoc string, forceSimple bool, opts *jsObj) *polyNumberFormatter {
	f := &polyNumberFormatter{}
	if p := opts.get("padTo"); truthy(p) {
		f.padTo = toNumber(p)
	}
	f.floor = truthy(opts.get("floor"))
	others := 0
	for _, k := range opts.keys {
		if k != "padTo" && k != "floor" {
			others++
		}
	}
	if !forceSimple || others > 0 {
		no := intl.NumberOptions{UseGrouping: false}
		if v := opts.get("useGrouping"); !isUndef(v) {
			no.UseGrouping = truthy(v)
		}
		if toNumber(opts.get("padTo")) > 0 {
			no.MinimumIntegerDigits = int(toNumber(opts.get("padTo")))
		}
		if v := opts.get("signDisplay"); !isUndef(v) {
			no.SignDisplay = toJSString(v)
		}
		if v := opts.get("style"); !isUndef(v) {
			switch s := toJSString(v); s {
			case "unit":
				no.Unit = toJSString(opts.get("unit"))
				if no.Unit == "undefined" {
					throw("TypeError", "Unit is required with style: unit")
				}
			case "decimal":
			default:
				throwRange("Value " + s + " out of range for Intl.NumberFormat options property style")
			}
		}
		if v := opts.get("unitDisplay"); !isUndef(v) {
			no.UnitDisplay = toJSString(v)
			switch no.UnitDisplay {
			case "short", "narrow", "long":
			default:
				throwRange("Value " + no.UnitDisplay + " out of range for Intl.NumberFormat options property unitDisplay")
			}
		}
		inf, err := intl.NewNumberFormat(intlLoc, no)
		raiseIntl(err)
		f.inf = inf
	}
	return f
}

func (f *polyNumberFormatter) format(i float64) string {
	if f.inf != nil {
		fixed := i
		if f.floor {
			fixed = math.Floor(i)
		}
		return f.inf.Format(fixed)
	}
	var fixed float64
	if f.floor {
		fixed = math.Floor(i)
	} else {
		fixed = roundTo(i, 3, "round")
	}
	return padStart(fixed, int(f.padTo))
}

// ---------------------------------------------------------------------------------------------
// PolyDateFormatter

type polyDateFormatter struct {
	opts         *jsObj
	dt           *DateTime
	originalZone Zone
	dtf          *intl.DateTimeFormat
}

func newPolyDateFormatter(dt *DateTime, intlLoc string, opts *jsObj) *polyDateFormatter {
	f := &polyDateFormatter{opts: opts}
	var z any = Undefined
	switch {
	case truthy(opts.get("timeZone")):
		f.dt = dt
	case dt.Zone().Type() == "fixed":
		gmtOffset := -1 * (dt.Offset() / 60)
		offsetZ := "Etc/GMT" + numberToString(gmtOffset)
		if gmtOffset >= 0 {
			offsetZ = "Etc/GMT+" + numberToString(gmtOffset)
		}
		if dt.Offset() != 0 && IANAZoneCreate(offsetZ).valid {
			z = offsetZ
			f.dt = dt
		} else {
			z = "UTC"
			if dt.Offset() == 0 {
				f.dt = dt
			} else {
				f.dt = dt.SetZone("UTC").Plus(Obj{"minutes", dt.Offset()})
			}
			f.originalZone = dt.Zone()
		}
	case dt.Zone().Type() == "system":
		f.dt = dt
	case dt.Zone().Type() == "iana":
		f.dt = dt
		z = dt.Zone().Name()
	default:
		z = "UTC"
		f.dt = dt.SetZone("UTC").Plus(Obj{"minutes", dt.Offset()})
		f.originalZone = dt.Zone()
	}
	intlOpts := opts.clone()
	if !truthy(intlOpts.get("timeZone")) {
		intlOpts.set("timeZone", z)
	}
	dtf, err := intl.NewDateTimeFormat(intlLoc, intlOptionMap(intlOpts))
	raiseIntl(err)
	f.dtf = dtf
	return f
}

// intlOptionMap is the options object as Intl reads it: undefined entries are absent.
func intlOptionMap(o *jsObj) map[string]any {
	m := make(map[string]any, len(o.keys))
	for _, k := range o.keys {
		v := o.vals[k]
		if isUndef(v) {
			continue
		}
		if f, ok := asFloat(v); ok {
			v = f
		}
		m[k] = v
	}
	return m
}

func (f *polyDateFormatter) jsDate() float64 {
	if f.dt.IsValid() {
		return f.dt.ts
	}
	return math.NaN()
}

func (f *polyDateFormatter) format() string {
	if f.originalZone != nil {
		var b strings.Builder
		for _, p := range f.formatToParts() {
			b.WriteString(p.Value)
		}
		return b.String()
	}
	s, err := f.dtf.Format(f.jsDate())
	raiseIntl(err)
	return s
}

func (f *polyDateFormatter) formatToParts() []intl.Part {
	parts, err := f.dtf.FormatToParts(f.jsDate())
	raiseIntl(err)
	if f.originalZone == nil {
		return parts
	}
	out := make([]intl.Part, len(parts))
	for i, p := range parts {
		if p.Type == "timeZoneName" {
			name := f.originalZone.OffsetName(f.dt.ts, Obj{"locale", strOrNil(f.dt.Locale()), "format", f.opts.get("timeZoneName")})
			if name == nil {
				p.Value = "null"
			} else {
				p.Value = *name
			}
		}
		out[i] = p
	}
	return out
}

func (f *polyDateFormatter) resolvedOptions() intl.ResolvedDateTimeOptions {
	return f.dtf.ResolvedOptions()
}

// ---------------------------------------------------------------------------------------------
// PolyRelFormatter

type polyRelFormatter struct {
	numeric any
	style   any
	rtf     *intl.RelativeTimeFormat
}

func newPolyRelFormatter(intlLoc string, isEnglish bool, opts *jsObj) *polyRelFormatter {
	f := &polyRelFormatter{numeric: opts.get("numeric"), style: "long"}
	if v := opts.get("style"); opts.has("style") {
		f.style = v
	}
	if !isEnglish {
		style, numeric := "", ""
		if v := opts.get("style"); !isUndef(v) {
			style = toJSString(v)
		}
		if v := opts.get("numeric"); !isUndef(v) {
			numeric = toJSString(v)
		}
		rtf, err := intl.NewRelativeTimeFormat(intlLoc, style, numeric)
		raiseIntl(err)
		f.rtf = rtf
	}
	return f
}

func (f *polyRelFormatter) format(count float64, unit string) string {
	if f.rtf != nil {
		s, err := f.rtf.Format(count, unit)
		raiseIntl(err)
		return s
	}
	numeric := "always"
	if !isUndef(f.numeric) {
		numeric = toJSString(f.numeric)
	}
	return formatRelativeTime(unit, count, numeric, f.style != "long")
}
