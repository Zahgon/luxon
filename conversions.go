package luxon

// impl/digits.js and impl/conversions.js

import (
	"math"
	"strings"
)

// numberingSystemSources are the JavaScript regular-expression sources of one digit in each
// numbering system. hanidec's class contains "|", and latn is the ASCII \d.
var numberingSystemSources = map[string]string{
	"arab":     `[٠-٩]`,
	"arabext":  `[۰-۹]`,
	"bali":     `[᭐-᭙]`,
	"beng":     `[০-৯]`,
	"deva":     `[०-९]`,
	"fullwide": `[０-９]`,
	"gujr":     `[૦-૯]`,
	"hanidec":  "[〇|一|二|三|四|五|六|七|八|九]",
	"khmr":     `[០-៩]`,
	"knda":     `[೦-೯]`,
	"laoo":     `[໐-໙]`,
	"limb":     `[᥆-᥏]`,
	"mlym":     `[൦-൯]`,
	"mong":     `[᠐-᠙]`,
	"mymr":     `[၀-၉]`,
	"orya":     `[୦-୯]`,
	"tamldec":  `[௦-௯]`,
	"telu":     `[౦-౯]`,
	"thai":     `[๐-๙]`,
	"tibt":     `[༠-༩]`,
	"latn":     `\d`,
}

// numberingSystemsUTF16 are the code ranges parseDigits maps, in the original's key order.
// fullwide stops at U+FF17, so the fullwidth 8 and 9 are never read.
var numberingSystemsUTF16 = []struct {
	min, max rune
}{
	{1632, 1641}, {1776, 1785}, {6992, 7001}, {2534, 2543}, {2406, 2415}, {65296, 65303},
	{2790, 2799}, {6112, 6121}, {3302, 3311}, {3792, 3801}, {6470, 6479}, {3430, 3439},
	{6160, 6169}, {4160, 4169}, {2918, 2927}, {3046, 3055}, {3174, 3183}, {3664, 3673},
	{3872, 3881},
}

const hanidecChars = "〇一二三四五六七八九"

// parseDigits reads digits in any of the known numbering systems.
func parseDigits(str string) float64 {
	value := parseInt10(str)
	if !math.IsNaN(value) {
		return value
	}
	var b strings.Builder
	for _, c := range str {
		if c == '|' || strings.ContainsRune(hanidecChars, c) {
			b.WriteString(numberToString(float64(indexRune(hanidecChars, c))))
			continue
		}
		for _, r := range numberingSystemsUTF16 {
			if c >= r.min && c <= r.max {
				b.WriteString(numberToString(float64(c - r.min)))
			}
		}
	}
	return parseInt10(b.String())
}

func indexRune(s string, c rune) int {
	i := 0
	for _, r := range s {
		if r == c {
			return i
		}
		i++
	}
	return -1
}

func resetDigitRegexCache() {}

// digitRegex is the JavaScript regular-expression source for digits in loc's numbering system.
func digitRegex(loc *locale, appendSrc string) string {
	ns := "latn"
	if loc.numberingSystem != nil && *loc.numberingSystem != "" {
		ns = *loc.numberingSystem
	}
	src, ok := numberingSystemSources[ns]
	if !ok {
		src = "undefined"
	}
	return src + appendSrc
}

// ---------------------------------------------------------------------------------------------
// conversions

var (
	nonLeapLadder = []float64{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}
	leapLadder    = []float64{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335}
)

func unitOutOfRange(unit string, value any) *invalid {
	return newInvalid("unit out of range", strp("you specified "+toJSString(value)+" (of type "+typeOf(value)+") as a "+unit+", which is invalid"))
}

// dayOfWeek is the ISO weekday (Monday 1 to Sunday 7) of a proleptic Gregorian date.
func dayOfWeek(year, month, day float64) float64 {
	t := dateUTC(year, month-1, day, 0, 0, 0, 0)
	if year < 100 && year >= 0 {
		f := utcFields(t)
		t = setUTCFullYear(t, f.year-1900, f.month0, f.date)
	}
	js := utcFields(t).weekday
	if js == 0 {
		return 7
	}
	return js
}

func ladderAt(year, month float64) float64 {
	table := nonLeapLadder
	if isLeapYear(year) {
		table = leapLadder
	}
	i := month - 1
	if !isIntegerF(i) || i < 0 || int(i) >= len(table) {
		return math.NaN()
	}
	return table[int(i)]
}

func computeOrdinal(year, month, day float64) float64 { return day + ladderAt(year, month) }

// uncomputeOrdinal keeps the original's findIndex(i => i < ordinal): any positive ordinal
// becomes month 1 with the ordinal as its day, which Date.UTC later rolls over.
func uncomputeOrdinal(year, ordinal float64) (month, day float64) {
	table := nonLeapLadder
	if isLeapYear(year) {
		table = leapLadder
	}
	month0 := -1
	for i, v := range table {
		if v < ordinal {
			month0 = i
			break
		}
	}
	if month0 < 0 {
		return 0, math.NaN()
	}
	return float64(month0 + 1), ordinal - table[month0]
}

func isoWeekdayToLocal(isoWeekday, startOfWeek float64) float64 {
	return jsMod(isoWeekday-startOfWeek+7, 7) + 1
}

// timeFields are the time units carried along by the calendar conversions; JavaScript copies
// them with pick(), so a missing one stays undefined (NaN here).
type timeFields struct{ hour, minute, second, millisecond float64 }

type weekObj struct {
	weekYear, weekNumber, weekday float64
	timeFields
}

type ordinalObj struct {
	year, ordinal float64
	timeFields
}

func (c calObj) timeFields() timeFields { return timeFields{c.hour, c.minute, c.second, c.millisecond} }

func gregorianToWeek(g calObj, minDaysInFirstWeek, startOfWeek float64) weekObj {
	ordinal := computeOrdinal(g.year, g.month, g.day)
	weekday := isoWeekdayToLocal(dayOfWeek(g.year, g.month, g.day), startOfWeek)
	weekNumber := math.Floor((ordinal - weekday + 14 - minDaysInFirstWeek) / 7)
	var weekYear float64
	if weekNumber < 1 {
		weekYear = g.year - 1
		weekNumber = weeksInWeekYear(weekYear, minDaysInFirstWeek, startOfWeek)
	} else if weekNumber > weeksInWeekYear(g.year, minDaysInFirstWeek, startOfWeek) {
		weekYear = g.year + 1
		weekNumber = 1
	} else {
		weekYear = g.year
	}
	return weekObj{weekYear, weekNumber, weekday, g.timeFields()}
}

func weekToGregorian(w weekObj, minDaysInFirstWeek, startOfWeek float64) calObj {
	weekdayOfJan4 := isoWeekdayToLocal(dayOfWeek(w.weekYear, 1, minDaysInFirstWeek), startOfWeek)
	yearInDays := daysInYear(w.weekYear)
	ordinal := float64(w.weekNumber*7) + w.weekday - weekdayOfJan4 - 7 + minDaysInFirstWeek
	var year float64
	if ordinal < 1 {
		year = w.weekYear - 1
		ordinal += daysInYear(year)
	} else if ordinal > yearInDays {
		year = w.weekYear + 1
		ordinal -= daysInYear(w.weekYear)
	} else {
		year = w.weekYear
	}
	month, day := uncomputeOrdinal(year, ordinal)
	return calObj{year, month, day, w.hour, w.minute, w.second, w.millisecond}
}

func gregorianToOrdinal(g calObj) ordinalObj {
	return ordinalObj{g.year, computeOrdinal(g.year, g.month, g.day), g.timeFields()}
}

func ordinalToGregorian(o ordinalObj) calObj {
	month, day := uncomputeOrdinal(o.year, o.ordinal)
	return calObj{o.year, month, day, o.hour, o.minute, o.second, o.millisecond}
}

// usesLocalWeekValues moves localWeekday/localWeekNumber/localWeekYear onto the ISO week keys
// and reports the week rules to use.
func usesLocalWeekValues(obj *jsObj, loc *locale) (minDaysInFirstWeek, startOfWeek float64) {
	hasLocal := obj.has("localWeekday") || obj.has("localWeekNumber") || obj.has("localWeekYear")
	if !hasLocal {
		return 4, 1
	}
	if obj.has("weekday") || obj.has("weekNumber") || obj.has("weekYear") {
		throwConflictingSpecification("Cannot mix locale-based week fields with ISO-based week fields")
	}
	if obj.has("localWeekday") {
		obj.set("weekday", obj.get("localWeekday"))
	}
	if obj.has("localWeekNumber") {
		obj.set("weekNumber", obj.get("localWeekNumber"))
	}
	if obj.has("localWeekYear") {
		obj.set("weekYear", obj.get("localWeekYear"))
	}
	obj.del("localWeekday")
	obj.del("localWeekNumber")
	obj.del("localWeekYear")
	return loc.getMinDaysInFirstWeek(), loc.getStartOfWeek()
}

func hasInvalidWeekData(w weekObj, minDaysInFirstWeek, startOfWeek float64) *invalid {
	switch {
	case !isIntegerF(w.weekYear):
		return unitOutOfRange("weekYear", w.weekYear)
	case !integerBetween(w.weekNumber, 1, weeksInWeekYear(w.weekYear, minDaysInFirstWeek, startOfWeek)):
		return unitOutOfRange("week", w.weekNumber)
	case !integerBetween(w.weekday, 1, 7):
		return unitOutOfRange("weekday", w.weekday)
	}
	return nil
}

func hasInvalidOrdinalData(o ordinalObj) *invalid {
	switch {
	case !isIntegerF(o.year):
		return unitOutOfRange("year", o.year)
	case !integerBetween(o.ordinal, 1, daysInYear(o.year)):
		return unitOutOfRange("ordinal", o.ordinal)
	}
	return nil
}

// hasInvalidGregorianData validates year, month and day, which may be any JavaScript value
// (DateTime.local and DateTime.utc pass their arguments through uncoerced).
func hasInvalidGregorianData(year, month, day any) *invalid {
	switch {
	case !integerBetween(year, math.Inf(-1), math.Inf(1)):
		return unitOutOfRange("year", year)
	case !integerBetween(month, 1, 12):
		return unitOutOfRange("month", month)
	case !integerBetween(day, 1, daysInMonth(toNumber(year), toNumber(month))):
		return unitOutOfRange("day", day)
	}
	return nil
}

func hasInvalidTimeData(hour, minute, second, millisecond any) *invalid {
	validHour := integerBetween(hour, 0, 23) || (strictNum(hour, 24) && strictNum(minute, 0) && strictNum(second, 0) && strictNum(millisecond, 0))
	switch {
	case !validHour:
		return unitOutOfRange("hour", hour)
	case !integerBetween(minute, 0, 59):
		return unitOutOfRange("minute", minute)
	case !integerBetween(second, 0, 59):
		return unitOutOfRange("second", second)
	case !integerBetween(millisecond, 0, 999):
		return unitOutOfRange("millisecond", millisecond)
	}
	return nil
}

// strictNum is v === n for a number n.
func strictNum(v any, n float64) bool {
	f, ok := asFloat(v)
	return ok && f == n
}

func (c calObj) invalidData() *invalid {
	if inv := hasInvalidGregorianData(c.year, c.month, c.day); inv != nil {
		return inv
	}
	return hasInvalidTimeData(c.hour, c.minute, c.second, c.millisecond)
}
