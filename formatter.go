package luxon

// impl/formatter.js

import (
	"math"
	"strings"

	"github.com/moment/luxon-go/internal/intl"
)

type fmtToken struct {
	literal bool
	val     string
}

var macroTokenToFormatOpts = map[string]Obj{
	"D": formatDateShort, "DD": formatDateMed, "DDD": formatDateFull, "DDDD": formatDateHuge,
	"t": formatTimeSimple, "tt": formatTimeWithSeconds, "ttt": formatTimeWithShortOffset, "tttt": formatTimeWithLongOffset,
	"T": formatTime24Simple, "TT": formatTime24WithSeconds, "TTT": formatTime24WithShortOffset, "TTTT": formatTime24WithLongOffset,
	"f": formatDatetimeShort, "ff": formatDatetimeMed, "fff": formatDatetimeFull, "ffff": formatDatetimeHuge,
	"F": formatDatetimeShortWithSecs, "FF": formatDatetimeMedWithSecs, "FFF": formatDatetimeFullWithSecs, "FFFF": formatDatetimeHugeWithSecs,
}

type formatter struct {
	opts      *jsObj
	loc       *locale
	systemLoc *locale
}

func newFormatter(loc *locale, opts *jsObj) *formatter {
	if opts == nil {
		opts = newObj()
	}
	return &formatter{opts: opts, loc: loc}
}

// parseFormat splits a format string into tokens and literals. It works on UTF-16 code units
// in the original; characters outside the BMP never repeat into a token, so runes suffice.
func parseFormat(fmt string) []fmtToken {
	var splits []fmtToken
	var current rune = -1
	currentFull := ""
	bracketed := false
	for _, c := range fmt {
		switch {
		case c == '\'':
			if len(currentFull) > 0 || bracketed {
				val := currentFull
				if val == "" {
					val = "'"
				}
				splits = append(splits, fmtToken{literal: bracketed || isAllWhitespace(currentFull), val: val})
			}
			current = -1
			currentFull = ""
			bracketed = !bracketed
		case bracketed:
			currentFull += string(c)
		case c == current:
			currentFull += string(c)
		default:
			if len(currentFull) > 0 {
				splits = append(splits, fmtToken{literal: isAllWhitespace(currentFull), val: currentFull})
			}
			currentFull = string(c)
			current = c
		}
	}
	if len(currentFull) > 0 {
		splits = append(splits, fmtToken{literal: bracketed || isAllWhitespace(currentFull), val: currentFull})
	}
	return splits
}

func stringifyTokens(splits []fmtToken, tokenToString func(string) string) string {
	var b strings.Builder
	for _, t := range splits {
		if t.literal {
			b.WriteString(t.val)
		} else {
			b.WriteString(tokenToString(t.val))
		}
	}
	return b.String()
}

func (f *formatter) formatWithSystemDefault(dt *DateTime, opts Obj) string {
	if f.systemLoc == nil {
		f.systemLoc = f.loc.redefaultToSystem(nil)
	}
	o := f.opts.clone().assign(objFrom(opts))
	return f.systemLoc.dtFormatter(dt, o).format()
}

func (f *formatter) dtFormatter(dt *DateTime, opts *jsObj) *polyDateFormatter {
	return f.loc.dtFormatter(dt, f.opts.clone().assign(opts))
}

func (f *formatter) formatDateTime(dt *DateTime) string { return f.dtFormatter(dt, nil).format() }

func (f *formatter) formatDateTimeParts(dt *DateTime) []intl.Part {
	return f.dtFormatter(dt, nil).formatToParts()
}

func (f *formatter) formatInterval(iv *Interval) string {
	df := f.dtFormatter(iv.s, nil)
	start, end := math.NaN(), math.NaN()
	if iv.s.IsValid() {
		start = iv.s.ts
	}
	if iv.e.IsValid() {
		end = iv.e.ts
	}
	s, err := df.dtf.FormatRange(start, end)
	raiseIntl(err)
	return s
}

func (f *formatter) resolvedOptions(dt *DateTime) intl.ResolvedDateTimeOptions {
	return f.dtFormatter(dt, nil).resolvedOptions()
}

// num formats a number, padded to p digits; signDisplay is "" when not given.
func (f *formatter) num(n float64, p int, signDisplay string) string {
	if truthy(f.opts.get("forceSimple")) {
		return padStart(n, p)
	}
	opts := f.opts.clone()
	if p > 0 {
		opts.set("padTo", float64(p))
	}
	if signDisplay != "" {
		opts.set("signDisplay", signDisplay)
	}
	return f.loc.numberFormatter(opts).format(n)
}

func orUndefined(s *string) string {
	if s == nil {
		return "undefined"
	}
	return *s
}

func orNull(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func (f *formatter) formatDateTimeFromString(dt *DateTime, fmt string) string {
	knownEnglish := f.loc.listingMode() == "en"
	useDateTimeFormatter := f.loc.outputCalendar != nil && *f.loc.outputCalendar != "" && *f.loc.outputCalendar != "gregory"
	str := func(opts Obj, extract string) string { return orNull(f.loc.extract(dt, objFrom(opts), extract)) }
	formatOffset := func(format string, allowZ any) string {
		if dt.IsOffsetFixed() != nil && *dt.IsOffsetFixed() && dt.Offset() == 0 && truthy(allowZ) {
			return "Z"
		}
		if dt.IsValid() {
			return dt.Zone().FormatOffset(dt.ts, format)
		}
		return ""
	}
	meridiem := func() string {
		if knownEnglish {
			return meridiemForDateTime(dt)
		}
		return str(Obj{"hour", "numeric", "hourCycle", "h12"}, "dayperiod")
	}
	month := func(length string, standalone bool) string {
		if knownEnglish {
			return orUndefined(monthForDateTime(dt, length))
		}
		if standalone {
			return str(Obj{"month", length}, "month")
		}
		return str(Obj{"month", length, "day", "numeric"}, "month")
	}
	weekday := func(length string, standalone bool) string {
		if knownEnglish {
			return orUndefined(weekdayForDateTime(dt, length))
		}
		if standalone {
			return str(Obj{"weekday", length}, "weekday")
		}
		return str(Obj{"weekday", length, "month", "long", "day", "numeric"}, "weekday")
	}
	era := func(length string) string {
		if knownEnglish {
			return orUndefined(eraForDateTime(dt, length))
		}
		return str(Obj{"era", length}, "era")
	}
	twoDigits := func(v float64) float64 { return stringToNumber(sliceLast(numberToString(v), 2)) }
	allowZ := f.opts.get("allowZ")
	tokenToString := func(token string) string {
		switch token {
		case "S":
			return f.num(dt.Millisecond(), 0, "")
		case "u", "SSS":
			return f.num(dt.Millisecond(), 3, "")
		case "s":
			return f.num(dt.Second(), 0, "")
		case "ss":
			return f.num(dt.Second(), 2, "")
		case "uu":
			return f.num(math.Floor(dt.Millisecond()/10), 2, "")
		case "uuu":
			return f.num(math.Floor(dt.Millisecond()/100), 0, "")
		case "m":
			return f.num(dt.Minute(), 0, "")
		case "mm":
			return f.num(dt.Minute(), 2, "")
		case "h", "hh":
			h := jsMod(dt.Hour(), 12)
			if h == 0 {
				h = 12
			}
			if token == "h" {
				return f.num(h, 0, "")
			}
			return f.num(h, 2, "")
		case "H":
			return f.num(dt.Hour(), 0, "")
		case "HH":
			return f.num(dt.Hour(), 2, "")
		case "Z":
			return formatOffset("narrow", allowZ)
		case "ZZ":
			return formatOffset("short", allowZ)
		case "ZZZ":
			return formatOffset("techie", allowZ)
		case "ZZZZ":
			return orNull(dt.Zone().OffsetName(dt.ts, Obj{"format", "short", "locale", f.loc.locale}))
		case "ZZZZZ":
			return orNull(dt.Zone().OffsetName(dt.ts, Obj{"format", "long", "locale", f.loc.locale}))
		case "z":
			return orNull(dt.ZoneName())
		case "a":
			return meridiem()
		case "d":
			if useDateTimeFormatter {
				return str(Obj{"day", "numeric"}, "day")
			}
			return f.num(dt.Day(), 0, "")
		case "dd":
			if useDateTimeFormatter {
				return str(Obj{"day", "2-digit"}, "day")
			}
			return f.num(dt.Day(), 2, "")
		case "c", "E":
			return f.num(dt.Weekday(), 0, "")
		case "ccc":
			return weekday("short", true)
		case "cccc":
			return weekday("long", true)
		case "ccccc":
			return weekday("narrow", true)
		case "EEE":
			return weekday("short", false)
		case "EEEE":
			return weekday("long", false)
		case "EEEEE":
			return weekday("narrow", false)
		case "L":
			if useDateTimeFormatter {
				return str(Obj{"month", "numeric", "day", "numeric"}, "month")
			}
			return f.num(dt.Month(), 0, "")
		case "LL":
			if useDateTimeFormatter {
				return str(Obj{"month", "2-digit", "day", "numeric"}, "month")
			}
			return f.num(dt.Month(), 2, "")
		case "LLL":
			return month("short", true)
		case "LLLL":
			return month("long", true)
		case "LLLLL":
			return month("narrow", true)
		case "M":
			if useDateTimeFormatter {
				return str(Obj{"month", "numeric"}, "month")
			}
			return f.num(dt.Month(), 0, "")
		case "MM":
			if useDateTimeFormatter {
				return str(Obj{"month", "2-digit"}, "month")
			}
			return f.num(dt.Month(), 2, "")
		case "MMM":
			return month("short", false)
		case "MMMM":
			return month("long", false)
		case "MMMMM":
			return month("narrow", false)
		case "y":
			if useDateTimeFormatter {
				return str(Obj{"year", "numeric"}, "year")
			}
			return f.num(dt.Year(), 0, "")
		case "yy":
			if useDateTimeFormatter {
				return str(Obj{"year", "2-digit"}, "year")
			}
			return f.num(twoDigits(dt.Year()), 2, "")
		case "yyyy":
			if useDateTimeFormatter {
				return str(Obj{"year", "numeric"}, "year")
			}
			return f.num(dt.Year(), 4, "")
		case "yyyyyy":
			if useDateTimeFormatter {
				return str(Obj{"year", "numeric"}, "year")
			}
			return f.num(dt.Year(), 6, "")
		case "G":
			return era("short")
		case "GG":
			return era("long")
		case "GGGGG":
			return era("narrow")
		case "kk":
			return f.num(twoDigits(dt.WeekYear()), 2, "")
		case "kkkk":
			return f.num(dt.WeekYear(), 4, "")
		case "W":
			return f.num(dt.WeekNumber(), 0, "")
		case "WW":
			return f.num(dt.WeekNumber(), 2, "")
		case "n":
			return f.num(dt.LocalWeekNumber(), 0, "")
		case "nn":
			return f.num(dt.LocalWeekNumber(), 2, "")
		case "ii":
			return f.num(twoDigits(dt.LocalWeekYear()), 2, "")
		case "iiii":
			return f.num(dt.LocalWeekYear(), 4, "")
		case "o":
			return f.num(dt.Ordinal(), 0, "")
		case "ooo":
			return f.num(dt.Ordinal(), 3, "")
		case "q":
			return f.num(dt.Quarter(), 0, "")
		case "qq":
			return f.num(dt.Quarter(), 2, "")
		case "X":
			return f.num(math.Floor(dt.ts/1000), 0, "")
		case "x":
			return f.num(dt.ts, 0, "")
		}
		if opts, ok := macroTokenToFormatOpts[token]; ok {
			return f.formatWithSystemDefault(dt, opts)
		}
		return token
	}
	return stringifyTokens(parseFormat(fmt), tokenToString)
}

// durationTokenField maps a duration format token to its unit, or "" for none.
func durationTokenField(token string) string {
	switch token[0] {
	case 'S':
		return "milliseconds"
	case 's':
		return "seconds"
	case 'm':
		return "minutes"
	case 'h':
		return "hours"
	case 'd':
		return "days"
	case 'w':
		return "weeks"
	case 'M':
		return "months"
	case 'y':
		return "years"
	}
	return ""
}

func (f *formatter) formatDurationFromString(dur *Duration, fmt string) string {
	signMode := f.opts.get("signMode")
	invertLargest := 1.0
	if signMode == "negativeLargestOnly" {
		invertLargest = -1
	}
	tokens := parseFormat(fmt)
	var units []any
	for _, t := range tokens {
		if !t.literal {
			if u := durationTokenField(t.val); u != "" {
				units = append(units, u)
			}
		}
	}
	collapsed := dur.ShiftTo(units...)
	isNegative := collapsed.ValueOf() < 0
	largestUnit := ""
	if collapsed.values != nil && len(collapsed.values.keys) > 0 {
		largestUnit = collapsed.values.keys[0]
	}
	tokenToString := func(token string) string {
		mapped := durationTokenField(token)
		if mapped == "" {
			return token
		}
		inversion := 1.0
		if isNegative && mapped != largestUnit {
			inversion = invertLargest
		}
		var signDisplay string
		switch {
		case signMode == "negativeLargestOnly" && mapped != largestUnit:
			signDisplay = "never"
		case signMode == "all":
			signDisplay = "always"
		default:
			signDisplay = "auto"
		}
		value := collapsed.Get(mapped) * inversion
		if signMode == "negativeLargestOnly" && mapped == largestUnit && isNegative && value == 0 {
			value = math.Copysign(0, -1)
		}
		return f.num(value, utf16Len(token), signDisplay)
	}
	return stringifyTokens(tokens, tokenToString)
}
