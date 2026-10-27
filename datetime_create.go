package luxon

import "math"

// DateTimeNow is DateTime.now().
func DateTimeNow() *DateTime { return newDateTime(dtConfig{}) }

// DateTimeLocal is DateTime.local(year, month, day, hour, minute, second, millisecond, opts).
func DateTimeLocal(args ...any) *DateTime {
	opts, rest, isNull := lastOpts(args)
	if isNull {
		throw("TypeError", "Cannot read properties of null (reading 'zone')")
	}
	return quickDT(unitArgs(rest), opts)
}

// DateTimeUtc is DateTime.utc(year, month, day, hour, minute, second, millisecond, opts).
func DateTimeUtc(args ...any) *DateTime {
	opts, rest, isNull := lastOpts(args)
	if isNull {
		throw("TypeError", "Cannot set properties of null (setting 'zone')")
	}
	opts.set("zone", utcInstance)
	return quickDT(unitArgs(rest), opts)
}

// unitArgs is {year, month, day, hour, minute, second, millisecond} from positional arguments.
func unitArgs(args []any) *jsObj {
	o := newObj()
	for i, u := range dtOrderedUnits {
		v := argAt(args, i)
		if f, ok := asFloat(v); ok {
			v = f
		}
		o.set(u, v)
	}
	return o
}

// objArg reads an object argument; nil stands for anything that is not an object.
func objArg(v any) *jsObj {
	switch o := v.(type) {
	case Obj:
		return objFrom(o)
	case *jsObj:
		return o
	}
	return nil
}

// DateTimeFromJSDate is DateTime.fromJSDate(date, options).
func DateTimeFromJSDate(date any, options ...any) *DateTime {
	ts := math.NaN()
	if d, ok := date.(JSDate); ok {
		ts = float64(d)
	}
	if math.IsNaN(ts) {
		return DateTimeInvalid("invalid input")
	}
	opts := optObj(options, 0)
	zone := normalizeZone(opts.get("zone"), Settings.DefaultZone())
	if !zone.IsValid() {
		return dateTimeInvalid(unsupportedZone(zone))
	}
	return newDateTime(dtConfig{ts: ts, hasTS: true, zone: zone, loc: newLocaleFromObject(opts)})
}

// DateTimeFromMillis is DateTime.fromMillis(milliseconds, options).
func DateTimeFromMillis(milliseconds any, options ...any) *DateTime {
	ms, ok := asFloat(milliseconds)
	if !ok {
		throwInvalidArgument("fromMillis requires a numerical input, but received a " + typeOf(milliseconds) +
			" with value " + toJSString(milliseconds))
	}
	if ms < -maxDate || ms > maxDate {
		return DateTimeInvalid("Timestamp out of range")
	}
	opts := optObj(options, 0)
	return newDateTime(dtConfig{ts: ms, hasTS: true, zone: normalizeZone(opts.get("zone"), Settings.DefaultZone()),
		loc: newLocaleFromObject(opts)})
}

// DateTimeFromSeconds is DateTime.fromSeconds(seconds, options).
func DateTimeFromSeconds(seconds any, options ...any) *DateTime {
	s, ok := asFloat(seconds)
	if !ok {
		throwInvalidArgument("fromSeconds requires a numerical input")
	}
	opts := optObj(options, 0)
	return newDateTime(dtConfig{ts: s * 1000, hasTS: true, zone: normalizeZone(opts.get("zone"), Settings.DefaultZone()),
		loc: newLocaleFromObject(opts)})
}

// DateTimeFromObject is DateTime.fromObject(obj, opts).
func DateTimeFromObject(obj any, opts ...any) *DateTime {
	return dateTimeFromObject(objArg(obj), optObj(opts, 0))
}

func dateTimeFromObject(obj *jsObj, opts *jsObj) *DateTime {
	if obj == nil {
		obj = newObj()
	}
	zoneToUse := normalizeZone(opts.get("zone"), Settings.DefaultZone())
	if !zoneToUse.IsValid() {
		return dateTimeInvalid(unsupportedZone(zoneToUse))
	}
	loc := newLocaleFromObject(opts)
	normalized := normalizeObject(obj, normalizeUnitWithLocalWeeks)
	minDays, startOfWeek := usesLocalWeekValues(normalized, loc)
	var tsNow, offsetProvis float64
	if v := opts.get("overrideNow"); !isNullish(v) {
		tsNow = toNumber(v)
	} else {
		tsNow = settingsNow()
	}
	if v := opts.get("specificOffset"); !isUndef(v) {
		offsetProvis = toNumber(v)
	} else {
		offsetProvis = zoneToUse.Offset(tsNow)
	}
	defined := func(k string) bool { return !isUndef(normalized.get(k)) }
	containsOrdinal := defined("ordinal")
	containsGregorMD := defined("month") || defined("day")
	containsGregor := defined("year") || containsGregorMD
	definiteWeekDef := truthy(normalized.get("weekYear")) || truthy(normalized.get("weekNumber"))
	if (containsGregor || containsOrdinal) && definiteWeekDef {
		throwConflictingSpecification("Can't mix weekYear/weekNumber units with year/month/day or ordinals")
	}
	if containsGregorMD && containsOrdinal {
		throwConflictingSpecification("Can't mix ordinal dates with month/day")
	}
	useWeekData := definiteWeekDef || (truthy(normalized.get("weekday")) && !containsGregor)

	objNow := tsToObj(tsNow, offsetProvis)
	var units []string
	var defaults, now map[string]float64
	switch {
	case useWeekData:
		units, defaults = dtOrderedWeekUnits, defaultWeekUnitValues
		w := gregorianToWeek(objNow, minDays, startOfWeek)
		now = map[string]float64{"weekYear": w.weekYear, "weekNumber": w.weekNumber, "weekday": w.weekday,
			"hour": w.hour, "minute": w.minute, "second": w.second, "millisecond": w.millisecond}
	case containsOrdinal:
		units, defaults = dtOrderedOrdinalUnits, defaultOrdinalUnitValues
		od := gregorianToOrdinal(objNow)
		now = map[string]float64{"year": od.year, "ordinal": od.ordinal,
			"hour": od.hour, "minute": od.minute, "second": od.second, "millisecond": od.millisecond}
	default:
		units, defaults = dtOrderedUnits, defaultUnitValues
		now = map[string]float64{"year": objNow.year, "month": objNow.month, "day": objNow.day,
			"hour": objNow.hour, "minute": objNow.minute, "second": objNow.second, "millisecond": objNow.millisecond}
	}
	foundFirst := false
	for _, u := range units {
		switch {
		case defined(u):
			foundFirst = true
		case foundFirst:
			normalized.set(u, defaults[u])
		default:
			normalized.set(u, now[u])
		}
	}

	n := func(k string) float64 { return fnum(normalized.get(k)) }
	tf := timeFields{n("hour"), n("minute"), n("second"), n("millisecond")}
	week := weekObj{n("weekYear"), n("weekNumber"), n("weekday"), tf}
	ord := ordinalObj{n("year"), n("ordinal"), tf}
	var inv *invalid
	switch {
	case useWeekData:
		inv = hasInvalidWeekData(week, minDays, startOfWeek)
	case containsOrdinal:
		inv = hasInvalidOrdinalData(ord)
	default:
		inv = hasInvalidGregorianData(normalized.get("year"), normalized.get("month"), normalized.get("day"))
	}
	if inv == nil {
		inv = hasInvalidTimeData(normalized.get("hour"), normalized.get("minute"), normalized.get("second"), normalized.get("millisecond"))
	}
	if inv != nil {
		return dateTimeInvalid(inv)
	}
	var gregorian calObj
	switch {
	case useWeekData:
		gregorian = weekToGregorian(week, minDays, startOfWeek)
	case containsOrdinal:
		gregorian = ordinalToGregorian(ord)
	default:
		gregorian = calFrom(normalized)
	}
	tsFinal, offsetFinal, wasHole := objToTS(gregorian, offsetProvis, zoneToUse)
	inst := newDateTime(dtConfig{ts: tsFinal, hasTS: true, zone: zoneToUse, o: offsetFinal, hasO: true, loc: loc, wasHole: wasHole})
	if truthy(normalized.get("weekday")) && containsGregor && !strictNum(obj.get("weekday"), inst.Weekday()) {
		return DateTimeInvalid("mismatched weekday", "you can't specify both a weekday of "+
			toJSString(normalized.get("weekday"))+" and a date of "+orNull(inst.ToISO()))
	}
	if !inst.IsValid() {
		return dateTimeInvalid(inst.invalid)
	}
	return inst
}

// DateTimeFromISO is DateTime.fromISO(text, opts).
func DateTimeFromISO(text string, opts ...any) *DateTime {
	vals, parsedZone := parseISODate(text)
	return parseDataToDateTime(vals, parsedZone, optObj(opts, 0), "ISO 8601", text, Undefined)
}

// DateTimeFromRFC2822 is DateTime.fromRFC2822(text, opts).
func DateTimeFromRFC2822(text string, opts ...any) *DateTime {
	vals, parsedZone := parseRFC2822Date(text)
	return parseDataToDateTime(vals, parsedZone, optObj(opts, 0), "RFC 2822", text, Undefined)
}

// DateTimeFromHTTP is DateTime.fromHTTP(text, opts).
func DateTimeFromHTTP(text string, opts ...any) *DateTime {
	vals, parsedZone := parseHTTPDate(text)
	return parseDataToDateTime(vals, parsedZone, optObj(opts, 0), "HTTP", text, Undefined)
}

// DateTimeFromSQL is DateTime.fromSQL(text, opts).
func DateTimeFromSQL(text string, opts ...any) *DateTime {
	vals, parsedZone := parseSQL(text)
	return parseDataToDateTime(vals, parsedZone, optObj(opts, 0), "SQL", text, Undefined)
}

// formatLocale is Locale.fromOpts({locale, numberingSystem, defaultToEN: true}) from parse options.
func formatLocale(o *jsObj) *locale {
	loc, ns := o.get("locale"), o.get("numberingSystem")
	if isUndef(loc) {
		loc = nil
	}
	if isUndef(ns) {
		ns = nil
	}
	return newLocaleFromOpts(objOf("locale", loc, "numberingSystem", ns, "defaultToEN", true))
}

// DateTimeFromFormat is DateTime.fromFormat(text, fmt, opts).
func DateTimeFromFormat(text, format string, opts ...any) *DateTime {
	o := optObj(opts, 0)
	vals, parsedZone, specificOffset, reason := parseFromTokens(formatLocale(o), text, format)
	if reason != nil {
		return DateTimeInvalid(*reason)
	}
	return parseDataToDateTime(vals, parsedZone, o, "format "+format, text, specificOffset)
}

// DateTimeFromString is DateTime.fromString, an alias of fromFormat.
func DateTimeFromString(text, format string, opts ...any) *DateTime {
	return DateTimeFromFormat(text, format, opts...)
}

// DateTimeFromFormatExplain is DateTime.fromFormatExplain(text, fmt, options).
func DateTimeFromFormatExplain(text, format string, opts ...any) Obj {
	return newTokenParser(formatLocale(optObj(opts, 0)), format).ExplainFromTokens(text)
}

// DateTimeFromStringExplain is DateTime.fromStringExplain, an alias of fromFormatExplain.
func DateTimeFromStringExplain(text, format string, opts ...any) Obj {
	return DateTimeFromFormatExplain(text, format, opts...)
}

// DateTimeBuildFormatParser is DateTime.buildFormatParser(fmt, options).
func DateTimeBuildFormatParser(format string, opts ...any) *TokenParser {
	return newTokenParser(formatLocale(optObj(opts, 0)), format)
}

// DateTimeFromFormatParser is DateTime.fromFormatParser(text, formatParser, opts).
func DateTimeFromFormatParser(text string, parser *TokenParser, opts ...any) *DateTime {
	o := optObj(opts, 0)
	localeToUse := formatLocale(o)
	if parser == nil {
		throw("TypeError", "Cannot read properties of null (reading 'locale')")
	}
	if !localeToUse.equals(parser.locale) {
		throwInvalidArgument("fromFormatParser called with a locale of " + localeToUse.String() +
			", but the format parser was created for " + parser.locale.String())
	}
	e := parser.explain(text)
	if e.invalidReason != nil {
		return DateTimeInvalid(*e.invalidReason)
	}
	return parseDataToDateTime(e.result, e.zone, o, "format "+parser.format, text, e.specificOffset)
}

// DateTimeInvalid is DateTime.invalid(reason, explanation).
func DateTimeInvalid(reason string, explanation ...any) *DateTime {
	if reason == "" {
		throwInvalidArgument("need to specify a reason the DateTime is invalid")
	}
	var ex *string
	if e := argAt(explanation, 0); !isNullish(e) {
		s := toJSString(e)
		ex = &s
	}
	return dateTimeInvalid(newInvalid(reason, ex))
}

// DateTimeIsDateTime is DateTime.isDateTime(o).
func DateTimeIsDateTime(o any) bool {
	dt, ok := o.(*DateTime)
	return ok && dt != nil
}

// DateTimeParseFormatForOpts is DateTime.parseFormatForOpts(formatOpts, localeOpts); nil is null.
func DateTimeParseFormatForOpts(formatOpts Obj, localeOpts ...any) *string {
	tokens := formatOptsToTokens(formatOpts, newLocaleFromObject(optObj(localeOpts, 0)))
	if tokens == nil {
		return nil
	}
	s := ""
	for _, t := range tokens {
		if t != nil {
			s += t.val
		}
	}
	return &s
}

// DateTimeExpandFormat is DateTime.expandFormat(fmt, localeOpts).
func DateTimeExpandFormat(format string, localeOpts ...any) string {
	s := ""
	for _, t := range expandMacroTokens(parseFormat(format), newLocaleFromObject(optObj(localeOpts, 0))) {
		s += t.val
	}
	return s
}

// DateTimeMin is DateTime.min(...dateTimes); nil is undefined (no arguments).
func DateTimeMin(dateTimes ...*DateTime) *DateTime {
	for _, d := range dateTimes {
		if d == nil {
			throwInvalidArgument("min requires all arguments be DateTimes")
		}
	}
	return bestBy(dateTimes, math.Min)
}

// DateTimeMax is DateTime.max(...dateTimes); nil is undefined (no arguments).
func DateTimeMax(dateTimes ...*DateTime) *DateTime {
	for _, d := range dateTimes {
		if d == nil {
			throwInvalidArgument("max requires all arguments be DateTimes")
		}
	}
	return bestBy(dateTimes, math.Max)
}

// friendlyDateTime accepts a DateTime, a JSDate or an object of units.
func friendlyDateTime(v any) *DateTime {
	switch x := v.(type) {
	case *DateTime:
		if x != nil {
			return x
		}
		v = nil
	case JSDate:
		return DateTimeFromJSDate(x)
	case Obj, *jsObj:
		return DateTimeFromObject(x)
	}
	if isNumber(v) {
		return DateTimeFromJSDate(v)
	}
	throwInvalidArgument("Unknown datetime argument: " + toJSString(v) + ", of type " + typeOf(v))
	return nil
}
