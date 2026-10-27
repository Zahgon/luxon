package luxon

import "math"

const invalidDateTimeString = "Invalid DateTime"

// maxDate bounds the time value of a JavaScript Date.
const maxDate = 8.64e15

// JSDate is a JavaScript Date, carried as its time value in milliseconds (NaN when invalid).
type JSDate float64

// DateTime is an immutable instant in a zone, with a locale.
type DateTime struct {
	ts            float64
	zone          Zone
	loc           *locale
	invalid       *invalid
	weekData      *weekObj
	localWeekData *weekObj
	c             *calObj
	o             float64
	wasHole       bool
}

// dtState is the snapshot clone() hands the constructor as config.old.
type dtState struct {
	ts   float64
	zone Zone
	c    *calObj
	o    float64
}

// dtConfig is the constructor's config object; hasTS and hasO mark a defined ts and o.
type dtConfig struct {
	ts      float64
	hasTS   bool
	zone    Zone
	o       float64
	hasO    bool
	loc     *locale
	invalid *invalid
	wasHole bool
	old     *dtState
}

// dtAlts are the properties clone() overrides; nil leaves one as it is.
type dtAlts struct {
	ts      *float64
	zone    Zone
	o       *float64
	loc     *locale
	wasHole *bool
}

func unsupportedZone(zone Zone) *invalid {
	return newInvalid("unsupported zone", strp(`the zone "`+zone.Name()+`" is not supported`))
}

func newDateTime(config dtConfig) *DateTime {
	zone := config.zone
	if zone == nil {
		zone = Settings.DefaultZone()
	}
	inv := config.invalid
	if inv == nil && config.hasTS && math.IsNaN(config.ts) {
		inv = newInvalid("invalid input", nil)
	}
	if inv == nil && !zone.IsValid() {
		inv = unsupportedZone(zone)
	}
	dt := &DateTime{zone: zone, o: math.NaN(), wasHole: config.wasHole}
	if config.hasTS {
		dt.ts = config.ts
	} else {
		dt.ts = settingsNow()
	}
	if inv == nil {
		if old := config.old; old != nil && old.ts == dt.ts && old.zone.Equals(zone) {
			dt.c, dt.o = old.c, old.o
		} else {
			var ot float64
			if config.hasO && old == nil {
				ot = config.o
			} else {
				ot = zone.Offset(dt.ts)
			}
			c := tsToObj(dt.ts, ot)
			if math.IsNaN(c.year) {
				inv = newInvalid("invalid input", nil)
			} else {
				dt.c, dt.o = &c, ot
			}
		}
	}
	dt.invalid = inv
	dt.loc = config.loc
	if dt.loc == nil {
		dt.loc = createLocale(nil, nil, nil, nil, false)
	}
	return dt
}

// clone is clone(inst, alts): a new DateTime from the current state, the overrides and old.
func (dt *DateTime) clone(alts dtAlts) *DateTime {
	config := dtConfig{ts: dt.ts, hasTS: true, zone: dt.zone, o: dt.o, hasO: true, loc: dt.loc,
		invalid: dt.invalid, wasHole: dt.wasHole, old: &dtState{dt.ts, dt.zone, dt.c, dt.o}}
	if alts.ts != nil {
		config.ts = *alts.ts
	}
	if alts.zone != nil {
		config.zone = alts.zone
	}
	if alts.o != nil {
		config.o = *alts.o
	}
	if alts.loc != nil {
		config.loc = alts.loc
	}
	if alts.wasHole != nil {
		config.wasHole = *alts.wasHole
	}
	return newDateTime(config)
}

// dateTimeInvalid is DateTime.invalid(invalid) for an Invalid already built.
func dateTimeInvalid(inv *invalid) *DateTime {
	if truthy(settingsThrowOnInvalid) {
		throwInvalidDateTime(inv)
	}
	return newDateTime(dtConfig{invalid: inv})
}

func tsToObj(ts, offset float64) calObj {
	f := utcFields(ts + float64(offset*60*1000))
	return calObj{f.year, f.month0 + 1, f.date, f.hours, f.minutes, f.seconds, f.ms}
}

func objToTS(obj calObj, offset float64, zone Zone) (ts, o float64, wasHole bool) {
	return fixOffset(objToLocalTS(obj), offset, zone)
}

// fixOffset finds the instant for a local timestamp, given a guess at the zone's offset.
func fixOffset(localTS, o float64, tz Zone) (float64, float64, bool) {
	utcGuess := localTS - float64(o*60*1000)
	o2 := tz.Offset(utcGuess)
	if o == o2 {
		return utcGuess, o, false
	}
	utcGuess -= float64((o2 - o) * 60 * 1000)
	o3 := tz.Offset(utcGuess)
	if o2 == o3 {
		return utcGuess, o2, false
	}
	return localTS - float64(math.Min(o2, o3)*60*1000), math.Max(o2, o3), true
}

// calFrom reads the Gregorian units of an object; a missing one is NaN.
func calFrom(o *jsObj) calObj {
	return calObj{fnum(o.get("year")), fnum(o.get("month")), fnum(o.get("day")), fnum(o.get("hour")),
		fnum(o.get("minute")), fnum(o.get("second")), fnum(o.get("millisecond"))}
}

// adjustTime applies a duration: calendar units on the local fields, the rest as milliseconds.
func adjustTime(inst *DateTime, dur *Duration) (ts, o float64, wasHole bool) {
	oPre := inst.o
	year := inst.c.year + math.Trunc(dur.Years())
	month := inst.c.month + math.Trunc(dur.Months()) + float64(math.Trunc(dur.Quarters())*3)
	c := *inst.c
	c.year, c.month = year, month
	c.day = math.Min(inst.c.day, daysInMonth(year, month)) + math.Trunc(dur.Days()) + float64(math.Trunc(dur.Weeks())*7)
	millisToAdd := DurationFromObject(Obj{
		"years", dur.Years() - math.Trunc(dur.Years()),
		"quarters", dur.Quarters() - math.Trunc(dur.Quarters()),
		"months", dur.Months() - math.Trunc(dur.Months()),
		"weeks", dur.Weeks() - math.Trunc(dur.Weeks()),
		"days", dur.Days() - math.Trunc(dur.Days()),
		"hours", dur.Hours(),
		"minutes", dur.Minutes(),
		"seconds", dur.Seconds(),
		"milliseconds", dur.Milliseconds(),
	}).As("milliseconds")
	ts, o, wasHole = fixOffset(objToLocalTS(c), oPre, inst.zone)
	if millisToAdd != 0 {
		ts += millisToAdd
		o = inst.zone.Offset(ts)
	}
	return ts, o, wasHole
}

func parseDataToDateTime(parsed *jsObj, parsedZone Zone, opts *jsObj, format, text string, specificOffset any) *DateTime {
	setZone, zone := opts.get("setZone"), opts.get("zone")
	if (parsed != nil && len(parsed.keys) != 0) || parsedZone != nil {
		var interpretationZone any = zone
		if parsedZone != nil {
			interpretationZone = parsedZone
		}
		o := opts.clone()
		o.set("zone", interpretationZone)
		o.set("specificOffset", specificOffset)
		inst := dateTimeFromObject(parsed, o)
		if truthy(setZone) {
			return inst
		}
		return inst.SetZone(zone)
	}
	return dateTimeInvalid(newInvalid("unparsable", strp(`the input "`+text+`" can't be parsed as `+format)))
}

// parseFromTokens is parseFromTokens(locale, input, format) from impl/tokenParser.js.
func parseFromTokens(loc *locale, input, format string) (*jsObj, Zone, any, *string) {
	e := newTokenParser(loc, format).explain(input)
	return e.result, e.zone, e.specificOffset, e.invalidReason
}

func toTechFormat(dt *DateTime, format string, allowZ bool) *string {
	if !dt.IsValid() {
		return nil
	}
	f := newFormatter(createLocale("en-US", nil, nil, nil, false), objOf("allowZ", allowZ, "forceSimple", true))
	s := f.formatDateTimeFromString(dt, format)
	return &s
}

func toISODate(o *DateTime, extended bool, precision string) string {
	year := o.c.year
	longFormat := year > 9999 || year < 0
	c := ""
	if longFormat && year >= 0 {
		c += "+"
	}
	if longFormat {
		c += padStart(year, 6)
	} else {
		c += padStart(year, 4)
	}
	if precision == "year" {
		return c
	}
	if extended {
		c += "-" + padStart(o.c.month, 2)
		if precision == "month" {
			return c
		}
		c += "-"
	} else {
		c += padStart(o.c.month, 2)
		if precision == "month" {
			return c
		}
	}
	return c + padStart(o.c.day, 2)
}

func toISOTime(o *DateTime, extended, suppressSeconds, suppressMilliseconds, includeOffset, extendedZone bool, precision string) string {
	showSeconds := !suppressSeconds || o.c.millisecond != 0 || o.c.second != 0
	c := ""
	switch precision {
	case "day", "month", "year":
	default:
		c += padStart(o.c.hour, 2)
		if precision == "hour" {
			break
		}
		if extended {
			c += ":" + padStart(o.c.minute, 2)
			if precision == "minute" {
				break
			}
			if showSeconds {
				c += ":" + padStart(o.c.second, 2)
			}
		} else {
			c += padStart(o.c.minute, 2)
			if precision == "minute" {
				break
			}
			if showSeconds {
				c += padStart(o.c.second, 2)
			}
		}
		if precision == "second" {
			break
		}
		if showSeconds && (!suppressMilliseconds || o.c.millisecond != 0) {
			c += "." + padStart(o.c.millisecond, 3)
		}
	}
	if includeOffset {
		sep := ""
		if extended {
			sep = ":"
		}
		switch {
		case derefBool(o.IsOffsetFixed()) && o.Offset() == 0 && !extendedZone:
			c += "Z"
		case o.o < 0:
			c += "-" + padStart(math.Trunc(-o.o/60), 2) + sep + padStart(math.Trunc(jsMod(-o.o, 60)), 2)
		default:
			c += "+" + padStart(math.Trunc(o.o/60), 2) + sep + padStart(math.Trunc(jsMod(o.o, 60)), 2)
		}
	}
	if extendedZone {
		c += "[" + o.zone.IanaName() + "]"
	}
	return c
}

func derefBool(b *bool) bool { return b != nil && *b }

var (
	dtOrderedUnits        = []string{"year", "month", "day", "hour", "minute", "second", "millisecond"}
	dtOrderedWeekUnits    = []string{"weekYear", "weekNumber", "weekday", "hour", "minute", "second", "millisecond"}
	dtOrderedOrdinalUnits = []string{"year", "ordinal", "hour", "minute", "second", "millisecond"}

	defaultUnitValues        = map[string]float64{"month": 1, "day": 1, "hour": 0, "minute": 0, "second": 0, "millisecond": 0}
	defaultWeekUnitValues    = map[string]float64{"weekNumber": 1, "weekday": 1, "hour": 0, "minute": 0, "second": 0, "millisecond": 0}
	defaultOrdinalUnitValues = map[string]float64{"ordinal": 1, "hour": 0, "minute": 0, "second": 0, "millisecond": 0}
)

var dtUnitNames = map[string]string{
	"year": "year", "years": "year", "month": "month", "months": "month", "day": "day", "days": "day",
	"hour": "hour", "hours": "hour", "minute": "minute", "minutes": "minute",
	"quarter": "quarter", "quarters": "quarter", "second": "second", "seconds": "second",
	"millisecond": "millisecond", "milliseconds": "millisecond", "weekday": "weekday", "weekdays": "weekday",
	"weeknumber": "weekNumber", "weeksnumber": "weekNumber", "weeknumbers": "weekNumber",
	"weekyear": "weekYear", "weekyears": "weekYear", "ordinal": "ordinal",
}

// lowerUnit is unit.toLowerCase(), with its TypeError for a value that is not a string.
func lowerUnit(unit any) string {
	switch u := unit.(type) {
	case string:
		return jsToLower(u)
	case undefinedType:
		throw("TypeError", "Cannot read properties of undefined (reading 'toLowerCase')")
	case nil:
		throw("TypeError", "Cannot read properties of null (reading 'toLowerCase')")
	}
	throw("TypeError", "unit.toLowerCase is not a function")
	return ""
}

// dtNormalizeUnit is datetime.js's normalizeUnit (singular names).
func dtNormalizeUnit(unit any) string {
	n, ok := dtUnitNames[lowerUnit(unit)]
	if !ok {
		throwInvalidUnit(unit)
	}
	return n
}

func normalizeUnitWithLocalWeeks(unit string) string {
	switch jsToLower(unit) {
	case "localweekday", "localweekdays":
		return "localWeekday"
	case "localweeknumber", "localweeknumbers":
		return "localWeekNumber"
	case "localweekyear", "localweekyears":
		return "localWeekYear"
	}
	return dtNormalizeUnit(unit)
}

var (
	zoneOffsetTs         *float64
	zoneOffsetGuessCache = map[string]float64{}
)

// guessOffsetForZone is the offset used to start the local-to-UTC search; it is computed once
// per zone, at the first Settings.now().
func guessOffsetForZone(zone Zone) float64 {
	if zoneOffsetTs == nil {
		t := settingsNow()
		zoneOffsetTs = &t
	}
	if zone.Type() != "iana" {
		return zone.Offset(*zoneOffsetTs)
	}
	name := zone.Name()
	guess, ok := zoneOffsetGuessCache[name]
	if !ok {
		guess = zone.Offset(*zoneOffsetTs)
		zoneOffsetGuessCache[name] = guess
	}
	return guess
}

// DateTimeResetCache is DateTime.resetCache().
func DateTimeResetCache() {
	zoneOffsetTs = nil
	zoneOffsetGuessCache = map[string]float64{}
}

func quickDT(obj *jsObj, opts *jsObj) *DateTime {
	zone := normalizeZone(opts.get("zone"), Settings.DefaultZone())
	if !zone.IsValid() {
		return dateTimeInvalid(unsupportedZone(zone))
	}
	config := dtConfig{zone: zone, loc: newLocaleFromObject(opts), hasTS: true}
	if !isUndef(obj.get("year")) {
		for _, u := range dtOrderedUnits {
			if isUndef(obj.get(u)) {
				obj.set(u, defaultUnitValues[u])
			}
		}
		inv := hasInvalidGregorianData(obj.get("year"), obj.get("month"), obj.get("day"))
		if inv == nil {
			inv = hasInvalidTimeData(obj.get("hour"), obj.get("minute"), obj.get("second"), obj.get("millisecond"))
		}
		if inv != nil {
			return dateTimeInvalid(inv)
		}
		config.ts, config.o, config.wasHole = objToTS(calFrom(obj), guessOffsetForZone(zone), zone)
		config.hasO = true
	} else {
		config.ts = settingsNow()
	}
	return newDateTime(config)
}

// lastOpts splits off a trailing options object; null counts as one, as typeof null is "object".
func lastOpts(args []any) (opts *jsObj, rest []any, isNull bool) {
	if n := len(args); n > 0 {
		switch o := args[n-1].(type) {
		case Obj:
			return objFrom(o), args[:n-1], false
		case *jsObj:
			if o != nil {
				return o, args[:n-1], false
			}
		case nil:
			return nil, args[:n-1], true
		}
	}
	return newObj(), args, false
}

func diffRelative(start, end *DateTime, opts *jsObj) string {
	round := opts.get("round")
	if isUndef(round) {
		round = true
	}
	rounding := opts.get("rounding")
	if isUndef(rounding) {
		rounding = "trunc"
	}
	calendary := truthy(opts.get("calendary"))
	format := func(c float64, unit string) string {
		digits, mode := 2.0, toJSString(rounding)
		if truthy(round) || calendary {
			digits = 0
		}
		if calendary {
			mode = "round"
		}
		c = roundTo(c, digits, mode)
		return end.loc.clone(opts).relFormatter(opts).format(c, unit)
	}
	differ := func(unit string) float64 {
		if calendary {
			if !end.HasSame(start, unit) {
				return end.StartOf(unit).Diff(start.StartOf(unit), unit).Get(unit)
			}
			return 0
		}
		return end.Diff(start, unit).Get(unit)
	}
	if u := opts.get("unit"); truthy(u) {
		return format(differ(toJSString(u)), toJSString(u))
	}
	units := anyList(opts.get("units"))
	for _, u := range units {
		unit := toJSString(u)
		if count := differ(unit); math.Abs(count) >= 1 {
			return format(count, unit)
		}
	}
	zero := 0.0
	if start.ValueOf() > end.ValueOf() {
		zero = math.Copysign(0, -1)
	}
	last := any(Undefined)
	if len(units) > 0 {
		last = units[len(units)-1]
	}
	return format(zero, toJSString(last))
}

// anyList reads a JavaScript array argument.
func anyList(v any) []any {
	switch l := v.(type) {
	case []any:
		return l
	case []string:
		out := make([]any, len(l))
		for i, s := range l {
			out[i] = s
		}
		return out
	}
	return nil
}

// bestBy is bestBy(arr, dt => dt.valueOf(), compare); nil is undefined.
func bestBy(arr []*DateTime, compare func(a, b float64) float64) *DateTime {
	var best *DateTime
	bestV := 0.0
	for i, next := range arr {
		v := next.ValueOf()
		if i == 0 || compare(bestV, v) != bestV {
			best, bestV = next, v
		}
	}
	return best
}
