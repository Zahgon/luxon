package luxon

// impl/util.js

import (
	"math"
	"regexp"
	"strings"

	"github.com/moment/luxon-go/internal/intl"
)

// weekSettings is a validated week configuration.
type weekSettings struct {
	firstDay    float64
	minimalDays float64
	weekend     []float64
}

func hasRelative() bool { return true }

func hasLocaleWeekInfo() bool { return true }

func integerBetween(thing any, bottom, top float64) bool {
	f, ok := asFloat(thing)
	return ok && isIntegerF(f) && f >= bottom && f <= top
}

// validateWeekSettings accepts nil/undefined (no settings) or an Obj.
func validateWeekSettings(settings any) *weekSettings {
	if isNullish(settings) {
		return nil
	}
	if ws, ok := settings.(*weekSettings); ok {
		return ws
	}
	o, ok := settings.(Obj)
	if !ok {
		if typeOf(settings) != "object" {
			throwInvalidArgument("Week settings must be an object")
		}
		throwInvalidArgument("Invalid week settings")
	}
	obj := objFrom(o)
	weekend, isArr := obj.get("weekend").([]any)
	if !integerBetween(obj.get("firstDay"), 1, 7) || !integerBetween(obj.get("minimalDays"), 1, 7) || !isArr {
		throwInvalidArgument("Invalid week settings")
	}
	for _, v := range weekend {
		if !integerBetween(v, 1, 7) {
			throwInvalidArgument("Invalid week settings")
		}
	}
	ws := &weekSettings{firstDay: toNumber(obj.get("firstDay")), minimalDays: toNumber(obj.get("minimalDays"))}
	for _, v := range weekend {
		ws.weekend = append(ws.weekend, toNumber(v))
	}
	return ws
}

// floorMod is x % n with the sign of n.
func floorMod(x, n float64) float64 { return x - float64(n*math.Floor(x/n)) }

// padStart formats a number, zero-padded to n characters after any minus sign.
func padStart(input float64, n int) string {
	if input < 0 {
		return "-" + padStartStr(numberToString(-input), n)
	}
	return padStartStr(numberToString(input), n)
}

// parseInteger is util.parseInteger: undefined for a missing or empty string.
func parseInteger(s *string) (float64, bool) {
	if s == nil || *s == "" {
		return 0, false
	}
	return parseInt10(*s), true
}

func parseFloating(s *string) (float64, bool) {
	if s == nil || *s == "" {
		return 0, false
	}
	return parseFloatJS(*s), true
}

func parseMillis(fraction *string) (float64, bool) {
	if fraction == nil || *fraction == "" {
		return 0, false
	}
	f := parseFloatJS("0."+*fraction) * 1000
	return math.Floor(f), true
}

func roundTo(number float64, digits float64, rounding string) float64 {
	factor := math.Pow(10, digits)
	switch rounding {
	case "expand":
		if number > 0 {
			return math.Ceil(number*factor) / factor
		}
		return math.Floor(number*factor) / factor
	case "trunc":
		return math.Trunc(number*factor) / factor
	case "round":
		return mathRound(number*factor) / factor
	case "floor":
		return math.Floor(number*factor) / factor
	case "ceil":
		return math.Ceil(number*factor) / factor
	}
	throwRange("Value rounding " + rounding + " is out of range")
	return 0
}

func snapFloatingPoint(val float64) float64 {
	r := mathRound(val)
	if math.Abs(val-r) < 4*2.220446049250313e-16*math.Max(1, math.Abs(r)) {
		return r
	}
	return val
}

func isLeapYear(year float64) bool {
	return jsMod(year, 4) == 0 && (jsMod(year, 100) != 0 || jsMod(year, 400) == 0)
}

func daysInYear(year float64) float64 {
	if isLeapYear(year) {
		return 366
	}
	return 365
}

var monthDays = []float64{31, math.NaN(), 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// daysInMonth returns NaN where the JavaScript returns undefined or null (non-integer months).
func daysInMonth(year, month float64) float64 {
	modMonth := floorMod(month-1, 12) + 1
	modYear := year + (month-modMonth)/12
	if modMonth == 2 {
		if isLeapYear(modYear) {
			return 29
		}
		return 28
	}
	if !isIntegerF(modMonth) || modMonth < 1 || modMonth > 12 {
		return math.NaN()
	}
	return monthDays[int(modMonth)-1]
}

// calObj is a calendar object {year, month, day, hour, minute, second, millisecond}.
type calObj struct {
	year, month, day, hour, minute, second, millisecond float64
}

// objToLocalTS converts a calendar object to a local timestamp (epoch with the offset baked in).
func objToLocalTS(o calObj) float64 {
	d := dateUTC(o.year, o.month-1, o.day, o.hour, o.minute, o.second, o.millisecond)
	if o.year < 100 && o.year >= 0 {
		d = setUTCFullYear(d, o.year, o.month-1, o.day)
	}
	return d
}

func firstWeekOffset(year, minDaysInFirstWeek, startOfWeek float64) float64 {
	fwdlw := isoWeekdayToLocal(dayOfWeek(year, 1, minDaysInFirstWeek), startOfWeek)
	return -fwdlw + minDaysInFirstWeek - 1
}

func weeksInWeekYear(weekYear, minDaysInFirstWeek, startOfWeek float64) float64 {
	weekOffset := firstWeekOffset(weekYear, minDaysInFirstWeek, startOfWeek)
	weekOffsetNext := firstWeekOffset(weekYear+1, minDaysInFirstWeek, startOfWeek)
	return (daysInYear(weekYear) - weekOffset + weekOffsetNext) / 7
}

func untruncateYear(year float64) float64 {
	if year > 99 {
		return year
	}
	if year > settingsTwoDigitCutoffYear {
		return 1900 + year
	}
	return 2000 + year
}

// parseZoneInfo returns the zone name ICU shows for ts, or nil.
func parseZoneInfo(ts float64, offsetFormat any, locale string, timeZone string) *string {
	style, _ := offsetFormat.(string)
	zone := timeZone
	if zone == "" {
		zone = intl.SystemTimeZone()
	}
	name, ok, err := intl.ZoneName(locale, zone, ts, style)
	raiseIntl(err)
	if !ok {
		return nil
	}
	return &name
}

// signedOffset('-5', '30') is -330.
func signedOffset(offHourStr, offMinuteStr *string) float64 {
	offHour := math.NaN()
	if offHourStr != nil {
		offHour = parseInt10(*offHourStr)
	}
	if math.IsNaN(offHour) {
		offHour = 0
	}
	offMin := 0.0
	if offMinuteStr != nil {
		if v := parseInt10(*offMinuteStr); !math.IsNaN(v) {
			offMin = v
		}
	}
	offMinSigned := offMin
	if offHour < 0 || isNegZero(offHour) {
		offMinSigned = -offMin
	}
	return float64(offHour*60) + offMinSigned
}

// asNumber coerces a unit value, throwing for booleans, empty strings and non-finite numbers.
func asNumber(value any) float64 {
	n := toNumber(value)
	_, isBool := value.(bool)
	if isBool || value == "" || math.IsNaN(n) || math.IsInf(n, 0) {
		throwInvalidArgument("Invalid unit value " + toJSString(value))
	}
	return n
}

// normalizeObject maps the keys of obj through normalizer and coerces the values, in key order.
func normalizeObject(obj *jsObj, normalizer func(string) string) *jsObj {
	normalized := newObj()
	for _, u := range obj.keys {
		v := obj.vals[u]
		if isNullish(v) {
			continue
		}
		normalized.set(normalizer(u), asNumber(v))
	}
	return normalized
}

// formatOffset renders an offset in minutes as +6, +06:00 or +0600.
func formatOffset(offset float64, format string) string {
	hours := math.Trunc(math.Abs(offset / 60))
	minutes := math.Trunc(math.Abs(jsMod(offset, 60)))
	sign := "-"
	if offset >= 0 {
		sign = "+"
	}
	switch format {
	case "short":
		return sign + padStart(hours, 2) + ":" + padStart(minutes, 2)
	case "narrow":
		s := sign + numberToString(hours)
		if minutes > 0 {
			s += ":" + numberToString(minutes)
		}
		return s
	case "techie":
		return sign + padStart(hours, 2) + padStart(minutes, 2)
	}
	throwRange("Value format " + format + " is out of range for property format")
	return ""
}

// raiseIntl turns an Intl failure into the exception JavaScript raises. A gap in the recorded
// ICU data is not a JavaScript exception: it is reported separately so that it can never pass
// for the original's behaviour.
func raiseIntl(err error) {
	if err == nil {
		return
	}
	switch e := err.(type) {
	case *intl.Error:
		throw(e.Name, e.Message)
	case *intl.Unsupported:
		panic(&Unsupported{What: e.What})
	}
	panic(err)
}

// Unsupported is raised when the embedded ICU data does not cover a locale or option set.
type Unsupported struct{ What string }

func (u *Unsupported) Error() string { return "luxon: no ICU data for " + u.What }

var reWhitespaceOnly = regexp.MustCompile(`^[\t\n\v\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}]+$`)

func isAllWhitespace(s string) bool { return reWhitespaceOnly.MatchString(s) }

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func indexOfStr(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

var _ = strings.TrimSpace
