package intl

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type dtfTemplate struct {
	RO    map[string]any `json:"ro"`
	Parts [][]string     `json:"parts"`
	Minus string         `json:"minus"`
	Lists map[string]int `json:"lists"`
	Year  string         `json:"year"`
	Frac  int            `json:"frac"`
	TZ    int            `json:"tz"`
}

type dtfContext struct {
	Bags map[string]json.RawMessage `json:"bags"`
}

type dtfData struct {
	Lists     []map[string]string    `json:"lists"`
	Templates []*dtfTemplate         `json:"templates"`
	Contexts  map[string]*dtfContext `json:"contexts"`
}

var (
	dtfOnce sync.Once
	dtfDat  dtfData
)

func dtfdata() *dtfData {
	dtfOnce.Do(func() { loadJSON("dtf", &dtfDat) })
	return &dtfDat
}

// Part is one element of formatToParts output.
type Part struct {
	Type  string
	Value string
}

// DateTimeFormat mirrors Intl.DateTimeFormat for the recorded locale and option combinations.
type DateTimeFormat struct {
	locale   string
	ctx      *dtfContext
	bagKey   string
	tpl      *dtfTemplate
	zone     string
	canon    string
	tzStyle  string
	calendar string
	ns       string
	digits   []string
}

var dtfFieldValues = map[string][]string{
	"weekday":      {"narrow", "short", "long"},
	"era":          {"narrow", "short", "long"},
	"year":         {"2-digit", "numeric"},
	"month":        {"2-digit", "numeric", "narrow", "short", "long"},
	"day":          {"2-digit", "numeric"},
	"dayPeriod":    {"narrow", "short", "long"},
	"hour":         {"2-digit", "numeric"},
	"minute":       {"2-digit", "numeric"},
	"second":       {"2-digit", "numeric"},
	"timeZoneName": {"short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"},
	"hourCycle":    {"h11", "h12", "h23", "h24"},
	"dateStyle":    {"full", "long", "medium", "short"},
	"timeStyle":    {"full", "long", "medium", "short"},
}

var dtfOrder = []string{"weekday", "era", "year", "month", "day", "dayPeriod", "hour", "minute", "second", "fractionalSecondDigits", "timeZoneName"}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// toOptionString is ECMA-402 GetOption's ToString for the values Luxon passes.
func toOptionString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	}
	return "undefined"
}

// bagKey canonicalizes DateTimeFormat options the way the recorder keyed them: the sorted
// [key, value] pairs of the date-time fields, as JSON.
func bagKey(opts map[string]any) (string, error) {
	type kv struct {
		k string
		v any
	}
	var pairs []kv
	hasDate, hasTime, hasStyle := false, false, false
	for _, k := range append(append([]string{}, dtfOrder...), "hour12", "hourCycle", "dateStyle", "timeStyle") {
		v, ok := opts[k]
		if !ok || v == nil {
			continue
		}
		switch k {
		case "hour12":
			b, isBool := v.(bool)
			if !isBool {
				return "", &Unsupported{What: "hour12 " + toOptionString(v)}
			}
			pairs = append(pairs, kv{k, b})
			continue
		case "fractionalSecondDigits":
			n, _ := strconv.ParseFloat(toOptionString(v), 64)
			if math.IsNaN(n) || n < 1 || n > 3 {
				return "", rangeError("fractionalSecondDigits value is out of range.")
			}
			pairs = append(pairs, kv{k, math.Floor(n)})
			hasTime = true
			continue
		}
		s := toOptionString(v)
		if !contains(dtfFieldValues[k], s) {
			return "", rangeError("Value " + s + " out of range for Intl.DateTimeFormat options property " + k)
		}
		pairs = append(pairs, kv{k, s})
		switch k {
		case "weekday", "year", "month", "day":
			hasDate = true
		case "dayPeriod", "hour", "minute", "second":
			hasTime = true
		case "dateStyle", "timeStyle":
			hasStyle = true
		}
	}
	if !hasDate && !hasTime && !hasStyle {
		pairs = append(pairs, kv{"year", "numeric"}, kv{"month", "numeric"}, kv{"day", "numeric"})
	}
	_, hasHour := opts["hour"]
	_, hasTimeStyle := opts["timeStyle"]
	if !hasHour && !hasTimeStyle {
		kept := pairs[:0]
		for _, p := range pairs {
			if p.k != "hour12" && p.k != "hourCycle" {
				kept = append(kept, p)
			}
		}
		pairs = kept
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].k < pairs[j].k })
	var b strings.Builder
	b.WriteByte('[')
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(p.k)
		vb, _ := json.Marshal(p.v)
		b.WriteByte('[')
		b.Write(kb)
		b.WriteByte(',')
		b.Write(vb)
		b.WriteByte(']')
	}
	b.WriteByte(']')
	return b.String(), nil
}

// NewDateTimeFormat mirrors new Intl.DateTimeFormat(locale, options). The timeZone option is
// read from opts["timeZone"]; an empty locale means the runtime default.
func NewDateTimeFormat(locale string, opts map[string]any) (*DateTimeFormat, error) {
	r, err := ResolveDateTimeFormat(locale)
	if err != nil {
		return nil, err
	}
	if locale == "" {
		locale = SystemLocale()
	}
	zone := SystemTimeZone()
	if tz, ok := opts["timeZone"]; ok && tz != nil {
		zone = toOptionString(tz)
	}
	canon, ok := CanonicalTimeZone(zone)
	if !ok {
		return nil, rangeError("Invalid time zone specified: " + zone)
	}
	key, err := bagKey(opts)
	if err != nil {
		return nil, err
	}
	d := dtfdata()
	ctx, ok := d.Contexts[locale]
	if !ok {
		return nil, &Unsupported{What: "date formats for locale " + locale}
	}
	raw, ok := ctx.Bags[key]
	if !ok {
		return nil, &Unsupported{What: "date format " + key + " in " + locale}
	}
	var idx int
	if err := json.Unmarshal(raw, &idx); err != nil {
		var t throwData
		if json.Unmarshal(raw, &t) == nil && len(t.Throw) == 2 {
			return nil, t.err()
		}
		return nil, &Unsupported{What: "date format " + key}
	}
	tpl := d.Templates[idx]
	f := &DateTimeFormat{locale: locale, ctx: ctx, bagKey: key, tpl: tpl, zone: zone, canon: canon}
	f.calendar, _ = tpl.RO["calendar"].(string)
	f.ns, _ = tpl.RO["numberingSystem"].(string)
	if f.calendar == "" {
		f.calendar = r.Calendar
	}
	if f.ns == "" {
		f.ns = r.NumberingSystem
	}
	if n, ok := locales().Num[locale]; ok && n.NS == f.ns {
		f.digits = n.Digits
	} else {
		f.digits = digitsFor(f.ns)
	}
	if s, ok := opts["timeZoneName"]; ok && s != nil {
		f.tzStyle = toOptionString(s)
	}
	return f, nil
}

func digitsFor(ns string) []string {
	for _, n := range locales().Num {
		if n.NS == ns {
			return n.Digits
		}
	}
	return []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
}

func (f *DateTimeFormat) localizeDigits(s string) string {
	if f.digits == nil || f.digits[0] == "0" {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteString(f.digits[r-'0'])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type instantFields struct {
	ms                   float64
	day                  int64
	cal                  calFields
	hour, minute, second int64
	milli                int64
}

func (f *DateTimeFormat) fieldsAt(ms float64) instantFields {
	sec := int64(math.Floor(ms / 1000))
	local := ms + float64(OffsetSeconds(f.canon, sec)*1000)
	day := int64(math.Floor(local / 86400000))
	rem := int64(local - float64(day)*86400000)
	return instantFields{
		ms: ms, day: day, cal: calendarFields(f.calendar, day),
		hour: rem / 3600000, minute: rem / 60000 % 60, second: rem / 1000 % 60, milli: rem % 1000,
	}
}

func (f *DateTimeFormat) list(typ string) map[string]string {
	i, ok := f.tpl.Lists[typ]
	if !ok {
		return nil
	}
	return dtfdata().Lists[i]
}

// partData supplies the field values of one rendering: the single-date template, or one range
// level with its own lists per part source.
type partData struct {
	list  func(typ string) map[string]string
	year  string
	minus string
	tz    int
}

func (f *DateTimeFormat) single() partData {
	return partData{list: f.list, year: f.tpl.Year, minus: f.tpl.Minus, tz: f.tpl.TZ}
}

func (f *DateTimeFormat) year(d partData, y int64) (string, error) {
	if d.year == "hebr" {
		return hebrewNumeral(y), nil
	}
	sign := ""
	if y < 0 {
		sign = d.minus
		y = -y
	}
	switch d.year {
	case "numeric", "":
		return sign + f.localizeDigits(strconv.FormatInt(y, 10)), nil
	case "2-digit":
		s := strconv.FormatInt(y%100, 10)
		if len(s) < 2 {
			s = "0" + s
		}
		return sign + f.localizeDigits(s), nil
	case "jpan":
		if sign == "" && y == 1 {
			return "元", nil
		}
		return sign + f.localizeDigits(strconv.FormatInt(y, 10)), nil
	}
	return "", &Unsupported{What: "year rule " + d.year}
}

func dayPeriodKey(x instantFields) string {
	z := func(v int64) string {
		if v == 0 {
			return "0"
		}
		return "x"
	}
	return strconv.FormatInt(x.hour, 10) + ":" + z(x.minute) + ":" + z(x.second) + ":" + z(x.milli)
}

func dayPeriod(l map[string]string, x instantFields) (string, bool) {
	if v, ok := l[dayPeriodKey(x)]; ok {
		return v, true
	}
	h := strconv.FormatInt(x.hour, 10) + ":"
	minuteZero := x.minute == 0
	var fallback string
	found := false
	for k, v := range l {
		if !strings.HasPrefix(k, h) {
			continue
		}
		if (k[len(h)] == '0') == minuteZero {
			return v, true
		}
		if x.hour != 12 && !found {
			fallback, found = v, true
		}
	}
	return fallback, found
}

func (f *DateTimeFormat) value(typ string, x instantFields, d partData) (string, bool, error) {
	lookup := func(key string) (string, bool, error) {
		v, ok := d.list(typ)[key]
		if !ok {
			return "", false, &Unsupported{What: typ + " " + key + " in " + f.locale}
		}
		return v, v != "", nil
	}
	switch typ {
	case "year":
		s, err := f.year(d, x.cal.Year)
		return s, s != "", err
	case "month":
		return lookup(x.cal.Month)
	case "day":
		return lookup(strconv.FormatInt(x.cal.Day, 10))
	case "weekday":
		return lookup(strconv.FormatInt(isoWeekday(x.day), 10))
	case "era":
		v, ok := d.list("era")[x.cal.Era]
		if !ok {
			// ICU's iso8601 calendar names an era only for narrow BCE dates: every other era is empty.
			if x.cal.Era == "" || f.calendar == "iso8601" {
				return "", false, nil
			}
			return "", false, &Unsupported{What: "era " + x.cal.Era + " in " + f.locale}
		}
		return v, v != "", nil
	case "hour":
		return lookup(strconv.FormatInt(x.hour, 10))
	case "minute":
		return lookup(strconv.FormatInt(x.minute, 10))
	case "second":
		return lookup(strconv.FormatInt(x.second, 10))
	case "dayPeriod":
		v, ok := dayPeriod(d.list("dayPeriod"), x)
		if !ok {
			return "", false, &Unsupported{What: "dayPeriod " + dayPeriodKey(x) + " in " + f.locale}
		}
		return v, v != "", nil
	case "fractionalSecond":
		s := strconv.FormatInt(x.milli+1000, 10)[1:]
		n := f.tpl.Frac
		if n < 1 || n > 3 {
			n = 3
		}
		return f.localizeDigits(s[:n]), true, nil
	case "timeZoneName":
		name, ok, err := zoneNameSlot(f.locale, f.zone, x.ms, d.tz)
		return name, ok && name != "", err
	}
	return "", false, &Unsupported{What: "part " + typ}
}

// render fills a part shape and returns each output part's source (empty for single dates);
// omitted parts let the literals around them merge, as ICU's field positions leave one literal
// run between fields.
func (f *DateTimeFormat) render(shape [][]string, pick func(i int, src string) (instantFields, partData)) ([]Part, []string, error) {
	out := make([]Part, 0, len(shape))
	var sources []string
	for i, p := range shape {
		src := ""
		if len(p) > 2 || (len(p) == 2 && p[0] != "literal") {
			src = p[1]
		}
		if p[0] == "literal" {
			if n := len(out); n > 0 && out[n-1].Type == "literal" {
				out[n-1].Value += p[len(p)-1]
				continue
			}
			out = append(out, Part{Type: "literal", Value: p[len(p)-1]})
			sources = append(sources, src)
			continue
		}
		x, d := pick(i, src)
		v, ok, err := f.value(p[0], x, d)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			out = append(out, Part{Type: p[0], Value: v})
			sources = append(sources, src)
		}
	}
	return out, sources, nil
}

func checkTime(ms float64) error {
	if !validTime(ms) {
		return knownError("dtfNaN")
	}
	return nil
}

// FormatToParts mirrors Intl.DateTimeFormat.prototype.formatToParts.
func (f *DateTimeFormat) FormatToParts(ms float64) ([]Part, error) {
	if err := checkTime(ms); err != nil {
		return nil, err
	}
	ms = math.Trunc(ms)
	x, d := f.fieldsAt(ms), f.single()
	parts, _, err := f.render(f.tpl.Parts, func(int, string) (instantFields, partData) { return x, d })
	return parts, err
}

// Format mirrors Intl.DateTimeFormat.prototype.format, which (unlike formatToParts) replaces
// U+202F NARROW NO-BREAK SPACE with a plain space.
func (f *DateTimeFormat) Format(ms float64) (string, error) {
	parts, err := f.FormatToParts(ms)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(joinParts(parts), "\u202f", " "), nil
}

func joinParts(parts []Part) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Value)
	}
	return b.String()
}

// ResolvedDateTimeOptions is the subset of resolvedOptions() Luxon reads.
type ResolvedDateTimeOptions struct {
	Locale          string
	Calendar        string
	NumberingSystem string
	TimeZone        string
	HourCycle       string
	Hour12          *bool
}

// ResolvedOptions mirrors Intl.DateTimeFormat.prototype.resolvedOptions.
func (f *DateTimeFormat) ResolvedOptions() ResolvedDateTimeOptions {
	r := ResolvedDateTimeOptions{TimeZone: f.canon, Calendar: f.calendar, NumberingSystem: f.ns}
	r.Locale, _ = f.tpl.RO["locale"].(string)
	r.HourCycle, _ = f.tpl.RO["hourCycle"].(string)
	if h, ok := f.tpl.RO["hour12"].(bool); ok {
		r.Hour12 = &h
	}
	return r
}

// IsValidTimeZone reports whether new Intl.DateTimeFormat("en-US", {timeZone}) succeeds.
func IsValidTimeZone(zone string) bool {
	_, ok := CanonicalTimeZone(zone)
	return ok
}
