package luxon

// impl/regexParser.js. The JavaScript pattern sources are kept verbatim and translated.

import (
	"regexp"
)

// extractor reads values from a match starting at cursor and returns them, a zone (or nil) and
// the next cursor.
type extractor func(m []*string, cursor int) (*jsObj, Zone, int)

type pattern struct {
	re *regexp.Regexp
	ex func(m []*string) (*jsObj, Zone)
}

func combineRegexes(sources ...string) *regexp.Regexp {
	full := ""
	for _, s := range sources {
		full += s
	}
	return compileJSRegex("^"+full+"$", "")
}

func combineExtractors(exs ...extractor) func(m []*string) (*jsObj, Zone) {
	return func(m []*string) (*jsObj, Zone) {
		merged := newObj()
		var zone Zone
		cursor := 1
		for _, ex := range exs {
			val, z, next := ex(m, cursor)
			merged.assign(val)
			if z != nil {
				zone = z
			}
			cursor = next
		}
		return merged, zone
	}
}

// parseWith runs the patterns in order; vals is nil when nothing matched.
func parseWith(s any, patterns ...pattern) (*jsObj, Zone) {
	if isNullish(s) {
		return nil, nil
	}
	str := toJSString(s)
	for _, p := range patterns {
		if m := jsExec(p.re, str); m != nil {
			return p.ex(m)
		}
	}
	return nil, nil
}

func group(m []*string, i int) *string {
	if i < len(m) {
		return m[i]
	}
	return nil
}

func simpleParse(keys ...string) extractor {
	return func(m []*string, cursor int) (*jsObj, Zone, int) {
		ret := newObj()
		i := 0
		for ; i < len(keys); i++ {
			if v, ok := parseInteger(group(m, cursor+i)); ok {
				ret.set(keys[i], v)
			} else {
				ret.set(keys[i], Undefined)
			}
		}
		return ret, nil, cursor + i
	}
}

const (
	ianaRegexSrc          = `[A-Za-z_+-]{1,256}(?::?\/[A-Za-z0-9_+-]{1,256}(?:\/[A-Za-z0-9_+-]{1,256})?)?`
	offsetRegexSrc        = `(?:([Zz])|([+-]\d\d)(?::?(\d\d))?)`
	isoExtendedZoneSrc    = `(?:` + offsetRegexSrc + `?(?:\[(` + ianaRegexSrc + `)\])?)?`
	isoTimeBaseRegexSrc   = `(\d\d)(?::?(\d\d)(?::?(\d\d)(?:[.,](\d{1,30}))?)?)?`
	isoTimeRegexSrc       = isoTimeBaseRegexSrc + isoExtendedZoneSrc
	isoTimeExtensionSrc   = `(?:[Tt]` + isoTimeRegexSrc + `)?`
	isoYmdRegexSrc        = `([+-]\d{6}|\d{4})(?:-?(\d\d)(?:-?(\d\d))?)?`
	isoWeekRegexSrc       = `(\d{4})-?W(\d\d)(?:-?(\d))?`
	isoOrdinalRegexSrc    = `(\d{4})-?(\d{3})`
	sqlYmdRegexSrc        = `(\d{4})-(\d\d)-(\d\d)`
	sqlTimeRegexSrc       = isoTimeBaseRegexSrc + ` ?(?:` + offsetRegexSrc + `|(` + ianaRegexSrc + `))?`
	sqlTimeExtensionSrc   = `(?: ` + sqlTimeRegexSrc + `)?`
	partialIsoIntervalEnd = `(?:(?:(\d\d)-)?(\d\d)?|(?:W(\d\d)-)?(\d)|(\d{3}))`
	isoDurationSrc        = `^-?P(?:(?:(-?\d{1,20}(?:\.\d{1,20})?)Y)?(?:(-?\d{1,20}(?:\.\d{1,20})?)M)?(?:(-?\d{1,20}(?:\.\d{1,20})?)W)?(?:(-?\d{1,20}(?:\.\d{1,20})?)D)?(?:T(?:(-?\d{1,20}(?:\.\d{1,20})?)H)?(?:(-?\d{1,20}(?:\.\d{1,20})?)M)?(?:(-?\d{1,20})(?:[.,](-?\d{1,20}))?S)?)?)$`
	rfc2822Src            = `^(?:(Mon|Tue|Wed|Thu|Fri|Sat|Sun),\s)?(\d{1,2})\s(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s(\d{2,4})\s(\d\d):(\d\d)(?::(\d\d))?\s(?:(UT|GMT|[ECMP][SD]T)|([Zz])|(?:([+-]\d\d)(\d\d)))$`
	rfc1123Src            = `^(Mon|Tue|Wed|Thu|Fri|Sat|Sun), (\d\d) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (\d{4}) (\d\d):(\d\d):(\d\d) GMT$`
	rfc850Src             = `^(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday), (\d\d)-(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)-(\d\d) (\d\d):(\d\d):(\d\d) GMT$`
	asciiSrc              = `^(Mon|Tue|Wed|Thu|Fri|Sat|Sun) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) ( \d|\d\d) (\d\d):(\d\d):(\d\d) (\d{4})$`
)

func intAt(m []*string, pos int, fallback any) any {
	s := group(m, pos)
	if s == nil {
		return fallback
	}
	if v, ok := parseInteger(s); ok {
		return v
	}
	return Undefined
}

func extractISOYmd(m []*string, cursor int) (*jsObj, Zone, int) {
	item := newObj()
	item.set("year", intAt(m, cursor, Undefined))
	item.set("month", intAt(m, cursor+1, 1.0))
	item.set("day", intAt(m, cursor+2, 1.0))
	return item, nil, cursor + 3
}

func extractISOTime(m []*string, cursor int) (*jsObj, Zone, int) {
	item := newObj()
	item.set("hours", intAt(m, cursor, 0.0))
	item.set("minutes", intAt(m, cursor+1, 0.0))
	item.set("seconds", intAt(m, cursor+2, 0.0))
	if v, ok := parseMillis(group(m, cursor+3)); ok {
		item.set("milliseconds", v)
	} else {
		item.set("milliseconds", Undefined)
	}
	return item, nil, cursor + 4
}

func nonEmpty(s *string) bool { return s != nil && *s != "" }

func extractISOOffset(m []*string, cursor int) (*jsObj, Zone, int) {
	local := !nonEmpty(group(m, cursor)) && !nonEmpty(group(m, cursor+1))
	fullOffset := signedOffset(group(m, cursor+1), group(m, cursor+2))
	var zone Zone
	if !local {
		zone = FixedOffsetZoneInstance(fullOffset)
	}
	return newObj(), zone, cursor + 3
}

func extractIANAZone(m []*string, cursor int) (*jsObj, Zone, int) {
	var zone Zone
	if s := group(m, cursor); nonEmpty(s) {
		zone = IANAZoneCreate(*s)
	}
	return newObj(), zone, cursor + 1
}

func extractISODuration(m []*string) (*jsObj, Zone) {
	s := *m[0]
	hasNegativePrefix := s[0] == '-'
	secondStr := group(m, 7)
	negativeSeconds := secondStr != nil && *secondStr != "" && (*secondStr)[0] == '-'
	maybeNegate := func(num float64, ok bool, force bool) any {
		if !ok {
			return Undefined
		}
		if force || (num != 0 && hasNegativePrefix && !isNaNf(num)) {
			return -num
		}
		return num
	}
	out := newObj()
	keys := []string{"years", "months", "weeks", "days", "hours", "minutes"}
	for i, k := range keys {
		v, ok := parseFloating(group(m, i+1))
		out.set(k, maybeNegate(v, ok, false))
	}
	sv, sok := parseFloating(secondStr)
	out.set("seconds", maybeNegate(sv, sok, secondStr != nil && *secondStr == "-0"))
	mv, mok := parseMillis(group(m, 8))
	out.set("milliseconds", maybeNegate(mv, mok, negativeSeconds))
	return out, nil
}

func isNaNf(f float64) bool { return f != f }

var obsOffsets = map[string]float64{
	"GMT": 0, "UT": 0, "EDT": -4 * 60, "EST": -5 * 60, "CDT": -5 * 60, "CST": -6 * 60,
	"MDT": -6 * 60, "MST": -7 * 60, "PDT": -7 * 60, "PST": -8 * 60,
}

func fromStrings(weekdayStr, yearStr, monthStr, dayStr, hourStr, minuteStr, secondStr *string) *jsObj {
	result := newObj()
	year, _ := parseInteger(yearStr)
	if yearStr != nil && utf16Len(*yearStr) == 2 {
		year = untruncateYear(year)
	}
	result.set("year", year)
	mon := ""
	if monthStr != nil {
		mon = *monthStr
	}
	result.set("month", float64(indexOfStr(monthsShort, mon)+1))
	day, _ := parseInteger(dayStr)
	result.set("day", day)
	hour, _ := parseInteger(hourStr)
	result.set("hour", hour)
	minute, _ := parseInteger(minuteStr)
	result.set("minute", minute)
	if nonEmpty(secondStr) {
		sec, _ := parseInteger(secondStr)
		result.set("second", sec)
	}
	if nonEmpty(weekdayStr) {
		if utf16Len(*weekdayStr) > 3 {
			result.set("weekday", float64(indexOfStr(weekdaysLong, *weekdayStr)+1))
		} else {
			result.set("weekday", float64(indexOfStr(weekdaysShort, *weekdayStr)+1))
		}
	}
	return result
}

func extractRFC2822(m []*string) (*jsObj, Zone) {
	result := fromStrings(group(m, 1), group(m, 4), group(m, 3), group(m, 2), group(m, 5), group(m, 6), group(m, 7))
	var offset float64
	switch {
	case nonEmpty(group(m, 8)):
		offset = obsOffsets[*group(m, 8)]
	case nonEmpty(group(m, 9)):
		offset = 0
	default:
		offset = signedOffset(group(m, 10), group(m, 11))
	}
	return result, NewFixedOffsetZone(offset)
}

var (
	reRFCComments = compileJSRegex(`\([^()]*\)|[\n\t]`, "")
	reRFCSpaces   = compileJSRegex(`(\s\s+)`, "")
)

func preprocessRFC2822(s string) string {
	s = reRFCComments.ReplaceAllString(s, " ")
	s = reRFCSpaces.ReplaceAllString(s, " ")
	return jsTrim(s)
}

func extractRFC1123Or850(m []*string) (*jsObj, Zone) {
	return fromStrings(group(m, 1), group(m, 4), group(m, 3), group(m, 2), group(m, 5), group(m, 6), group(m, 7)), utcInstance
}

func extractASCII(m []*string) (*jsObj, Zone) {
	return fromStrings(group(m, 1), group(m, 7), group(m, 2), group(m, 3), group(m, 4), group(m, 5), group(m, 6)), utcInstance
}

var (
	isoYmdWithTimeExtension     = combineRegexes(isoYmdRegexSrc, isoTimeExtensionSrc)
	isoWeekWithTimeExtension    = combineRegexes(isoWeekRegexSrc, isoTimeExtensionSrc)
	isoOrdinalWithTimeExtension = combineRegexes(isoOrdinalRegexSrc, isoTimeExtensionSrc)
	isoTimeCombined             = combineRegexes(isoTimeRegexSrc)
	isoIntervalEndDateTime      = combineRegexes(partialIsoIntervalEnd, isoTimeExtensionSrc)
	isoTimeOnly                 = compileJSRegex(`^T?`+isoTimeBaseRegexSrc+`$`, "")
	isoDuration                 = compileJSRegex(isoDurationSrc, "")
	rfc2822                     = compileJSRegex(rfc2822Src, "")
	rfc1123                     = compileJSRegex(rfc1123Src, "")
	rfc850                      = compileJSRegex(rfc850Src, "")
	asciiDate                   = compileJSRegex(asciiSrc, "")
	sqlYmdWithTimeExtension     = combineRegexes(sqlYmdRegexSrc, sqlTimeExtensionSrc)
	sqlTimeCombined             = combineRegexes(sqlTimeRegexSrc)

	extractISOWeekData    = simpleParse("weekYear", "weekNumber", "weekday")
	extractISOOrdinalData = simpleParse("year", "ordinal")

	extractISOYmdTimeAndOffset      = combineExtractors(extractISOYmd, extractISOTime, extractISOOffset, extractIANAZone)
	extractISOWeekTimeAndOffset     = combineExtractors(extractISOWeekData, extractISOTime, extractISOOffset, extractIANAZone)
	extractISOOrdinalDateAndTime    = combineExtractors(extractISOOrdinalData, extractISOTime, extractISOOffset, extractIANAZone)
	extractISOTimeAndOffset         = combineExtractors(extractISOTime, extractISOOffset, extractIANAZone)
	extractPartialIsoIntervalEnd    = simpleParse("month", "day", "weekNumber", "weekday", "ordinal")
	extractISOIntervalPartialDate   = combineExtractors(extractPartialIsoIntervalEnd, extractISOTime, extractISOOffset, extractIANAZone)
	extractISOTimeOnly              = combineExtractors(extractISOTime)
	extractISOTimeOffsetAndIANAZone = combineExtractors(extractISOTime, extractISOOffset, extractIANAZone)
)

func parseISODate(s any) (*jsObj, Zone) {
	return parseWith(s,
		pattern{isoYmdWithTimeExtension, extractISOYmdTimeAndOffset},
		pattern{isoWeekWithTimeExtension, extractISOWeekTimeAndOffset},
		pattern{isoOrdinalWithTimeExtension, extractISOOrdinalDateAndTime},
		pattern{isoTimeCombined, extractISOTimeAndOffset})
}

func parseISOIntervalEnd(s any) (*jsObj, Zone) {
	return parseWith(s,
		pattern{isoIntervalEndDateTime, extractISOIntervalPartialDate},
		pattern{isoYmdWithTimeExtension, extractISOYmdTimeAndOffset},
		pattern{isoWeekWithTimeExtension, extractISOWeekTimeAndOffset},
		pattern{isoOrdinalWithTimeExtension, extractISOOrdinalDateAndTime},
		pattern{isoTimeCombined, extractISOTimeAndOffset})
}

func parseRFC2822Date(s any) (*jsObj, Zone) {
	str, ok := s.(string)
	if !ok {
		if isNullish(s) {
			throw("TypeError", "Cannot read properties of "+toJSString(s)+" (reading 'replace')")
		}
		throw("TypeError", "s.replace is not a function")
	}
	return parseWith(preprocessRFC2822(str), pattern{rfc2822, extractRFC2822})
}

func parseHTTPDate(s any) (*jsObj, Zone) {
	return parseWith(s,
		pattern{rfc1123, extractRFC1123Or850},
		pattern{rfc850, extractRFC1123Or850},
		pattern{asciiDate, extractASCII})
}

func parseISODuration(s any) *jsObj {
	vals, _ := parseWith(s, pattern{isoDuration, extractISODuration})
	return vals
}

func parseISOTimeOnly(s any) *jsObj {
	vals, _ := parseWith(s, pattern{isoTimeOnly, extractISOTimeOnly})
	return vals
}

func parseSQL(s any) (*jsObj, Zone) {
	return parseWith(s,
		pattern{sqlYmdWithTimeExtension, extractISOYmdTimeAndOffset},
		pattern{sqlTimeCombined, extractISOTimeOffsetAndIANAZone})
}
