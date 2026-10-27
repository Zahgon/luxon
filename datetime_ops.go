package luxon

import "math"

// optDefault is a destructuring default: the option's value, or def when it is undefined.
func optDefault(o *jsObj, key string, def any) any {
	if v := o.get(key); !isUndef(v) {
		return v
	}
	return def
}

// ResolvedLocaleOptions is DateTime#resolvedLocaleOptions(opts).
func (dt *DateTime) ResolvedLocaleOptions(opts ...any) Obj {
	o := optObj(opts, 0)
	r := newFormatter(dt.loc.clone(o), o).resolvedOptions(dt)
	return Obj{"locale", r.Locale, "numberingSystem", r.NumberingSystem, "outputCalendar", r.Calendar}
}

// ToUTC is DateTime#toUTC(offset = 0, opts).
func (dt *DateTime) ToUTC(args ...any) *DateTime {
	offset := argAt(args, 0)
	if isUndef(offset) {
		offset = 0.0
	}
	return dt.SetZone(FixedOffsetZoneInstance(toNumber(offset)), argAt(args, 1))
}

// ToLocal is DateTime#toLocal().
func (dt *DateTime) ToLocal() *DateTime { return dt.SetZone(Settings.DefaultZone()) }

// SetZone is DateTime#setZone(zone, {keepLocalTime, keepCalendarTime}).
func (dt *DateTime) SetZone(zone any, opts ...any) *DateTime {
	z := normalizeZone(zone, Settings.DefaultZone())
	if z.Equals(dt.zone) {
		return dt
	}
	if !z.IsValid() {
		return dateTimeInvalid(unsupportedZone(z))
	}
	o := optObj(opts, 0)
	newTS, wasHole := dt.ts, false
	if truthy(o.get("keepLocalTime")) || truthy(o.get("keepCalendarTime")) {
		asObj := calFrom(objFrom(dt.ToObject()))
		newTS, _, wasHole = objToTS(asObj, z.Offset(dt.ts), z)
	}
	return dt.clone(dtAlts{ts: &newTS, zone: z, wasHole: &wasHole})
}

// Reconfigure is DateTime#reconfigure({locale, numberingSystem, outputCalendar, weekSettings}).
func (dt *DateTime) Reconfigure(opts ...any) *DateTime {
	o := optObj(opts, 0)
	loc := dt.loc.clone(objOf("locale", o.get("locale"), "numberingSystem", o.get("numberingSystem"),
		"outputCalendar", o.get("outputCalendar"), "weekSettings", o.get("weekSettings")))
	return dt.clone(dtAlts{loc: loc})
}

// SetLocale is DateTime#setLocale(locale).
func (dt *DateTime) SetLocale(locale any) *DateTime { return dt.Reconfigure(Obj{"locale", locale}) }

// Set is DateTime#set(values).
func (dt *DateTime) Set(values Obj) *DateTime {
	if !dt.IsValid() {
		return dt
	}
	normalized := normalizeObject(objFrom(values), normalizeUnitWithLocalWeeks)
	minDays, startOfWeek := usesLocalWeekValues(normalized, dt.loc)
	defined := func(k string) bool { return !isUndef(normalized.get(k)) }
	settingWeekStuff := defined("weekYear") || defined("weekNumber") || defined("weekday")
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
	over := func(k string, v *float64) {
		if defined(k) {
			*v = fnum(normalized.get(k))
		}
	}
	overTime := func(t *timeFields) {
		over("hour", &t.hour)
		over("minute", &t.minute)
		over("second", &t.second)
		over("millisecond", &t.millisecond)
	}
	var mixed calObj
	switch {
	case settingWeekStuff:
		w := gregorianToWeek(*dt.c, minDays, startOfWeek)
		over("weekYear", &w.weekYear)
		over("weekNumber", &w.weekNumber)
		over("weekday", &w.weekday)
		overTime(&w.timeFields)
		mixed = weekToGregorian(w, minDays, startOfWeek)
	case containsOrdinal:
		od := gregorianToOrdinal(*dt.c)
		over("year", &od.year)
		over("ordinal", &od.ordinal)
		overTime(&od.timeFields)
		mixed = ordinalToGregorian(od)
	default:
		mixed = *dt.c
		over("year", &mixed.year)
		over("month", &mixed.month)
		over("day", &mixed.day)
		over("hour", &mixed.hour)
		over("minute", &mixed.minute)
		over("second", &mixed.second)
		over("millisecond", &mixed.millisecond)
		if !defined("day") {
			mixed.day = math.Min(daysInMonth(mixed.year, mixed.month), mixed.day)
		}
	}
	ts, o, wasHole := objToTS(mixed, dt.o, dt.zone)
	return dt.clone(dtAlts{ts: &ts, o: &o, wasHole: &wasHole})
}

// Plus is DateTime#plus(duration): a Duration, a number of milliseconds or an object of units.
func (dt *DateTime) Plus(duration any) *DateTime {
	if !dt.IsValid() {
		return dt
	}
	ts, o, wasHole := adjustTime(dt, DurationFromDurationLike(duration))
	return dt.clone(dtAlts{ts: &ts, o: &o, wasHole: &wasHole})
}

// Minus is DateTime#minus(duration).
func (dt *DateTime) Minus(duration any) *DateTime {
	if !dt.IsValid() {
		return dt
	}
	ts, o, wasHole := adjustTime(dt, DurationFromDurationLike(duration).Negate())
	return dt.clone(dtAlts{ts: &ts, o: &o, wasHole: &wasHole})
}

// StartOf is DateTime#startOf(unit, {useLocaleWeeks}).
func (dt *DateTime) StartOf(unit any, opts ...any) *DateTime {
	if !dt.IsValid() {
		return dt
	}
	useLocaleWeeks := truthy(optObj(opts, 0).get("useLocaleWeeks"))
	o := newObj()
	normalizedUnit := DurationNormalizeUnit(unit)
	switch normalizedUnit {
	case "years":
		o.set("month", 1.0)
		fallthrough
	case "quarters", "months":
		o.set("day", 1.0)
		fallthrough
	case "weeks", "days":
		o.set("hour", 0.0)
		fallthrough
	case "hours":
		o.set("minute", 0.0)
		fallthrough
	case "minutes":
		o.set("second", 0.0)
		fallthrough
	case "seconds":
		o.set("millisecond", 0.0)
	}
	if normalizedUnit == "weeks" {
		if useLocaleWeeks {
			startOfWeek := dt.loc.getStartOfWeek()
			if dt.Weekday() < startOfWeek {
				o.set("weekNumber", dt.WeekNumber()-1)
			}
			o.set("weekday", startOfWeek)
		} else {
			o.set("weekday", 1.0)
		}
	}
	if normalizedUnit == "quarters" {
		q := math.Ceil(dt.Month() / 3)
		o.set("month", float64((q-1)*3)+1)
	}
	return dt.Set(o.toObj())
}

// EndOf is DateTime#endOf(unit, opts).
func (dt *DateTime) EndOf(unit any, opts ...any) *DateTime {
	if !dt.IsValid() {
		return dt
	}
	return dt.Plus(Obj{toJSString(unit), 1.0}).StartOf(unit, opts...).Minus(1.0)
}

// ToFormat is DateTime#toFormat(fmt, opts).
func (dt *DateTime) ToFormat(fmt string, opts ...any) string {
	if !dt.IsValid() {
		return invalidDateTimeString
	}
	return newFormatter(dt.loc.redefaultToEN(optObj(opts, 0)), nil).formatDateTimeFromString(dt, fmt)
}

// ToLocaleString is DateTime#toLocaleString(formatOpts = DATE_SHORT, opts).
func (dt *DateTime) ToLocaleString(args ...any) string {
	if !dt.IsValid() {
		return invalidDateTimeString
	}
	var formatOpts *jsObj
	switch f := argAt(args, 0).(type) {
	case undefinedType:
		formatOpts = objFrom(DateTimeDATE_SHORT())
	case Obj:
		formatOpts = objFrom(f)
	case *jsObj:
		formatOpts = f
	}
	return newFormatter(dt.loc.clone(optObj(args, 1)), formatOpts).formatDateTime(dt)
}

// ToLocaleParts is DateTime#toLocaleParts(opts): the {type, value} parts.
func (dt *DateTime) ToLocaleParts(opts ...any) []Obj {
	if !dt.IsValid() {
		return []Obj{}
	}
	o := optObj(opts, 0)
	parts := newFormatter(dt.loc.clone(o), o).formatDateTimeParts(dt)
	out := make([]Obj, len(parts))
	for i, p := range parts {
		out[i] = Obj{"type", p.Type, "value", p.Value}
	}
	return out
}

// ToISO is DateTime#toISO(opts); nil is null.
func (dt *DateTime) ToISO(opts ...any) *string {
	if !dt.IsValid() {
		return nil
	}
	o := optObj(opts, 0)
	precision := dtNormalizeUnit(optDefault(o, "precision", "milliseconds"))
	ext := optDefault(o, "format", "extended") == "extended"
	c := toISODate(dt, ext, precision)
	if indexOfStr(dtOrderedUnits, precision) >= 3 {
		c += "T"
	}
	c += toISOTime(dt, ext, truthy(optDefault(o, "suppressSeconds", false)),
		truthy(optDefault(o, "suppressMilliseconds", false)), truthy(optDefault(o, "includeOffset", true)),
		truthy(optDefault(o, "extendedZone", false)), precision)
	return &c
}

// ToISODate is DateTime#toISODate({format, precision}); nil is null.
func (dt *DateTime) ToISODate(opts ...any) *string {
	if !dt.IsValid() {
		return nil
	}
	o := optObj(opts, 0)
	s := toISODate(dt, optDefault(o, "format", "extended") == "extended", dtNormalizeUnit(optDefault(o, "precision", "day")))
	return &s
}

// ToISOWeekDate is DateTime#toISOWeekDate(); nil is null.
func (dt *DateTime) ToISOWeekDate() *string { return toTechFormat(dt, "kkkk-'W'WW-c", true) }

// ToISOTime is DateTime#toISOTime(opts); nil is null.
func (dt *DateTime) ToISOTime(opts ...any) *string {
	if !dt.IsValid() {
		return nil
	}
	o := optObj(opts, 0)
	precision := dtNormalizeUnit(optDefault(o, "precision", "milliseconds"))
	c := ""
	if truthy(optDefault(o, "includePrefix", false)) && indexOfStr(dtOrderedUnits, precision) >= 3 {
		c = "T"
	}
	c += toISOTime(dt, optDefault(o, "format", "extended") == "extended",
		truthy(optDefault(o, "suppressSeconds", false)), truthy(optDefault(o, "suppressMilliseconds", false)),
		truthy(optDefault(o, "includeOffset", true)), truthy(optDefault(o, "extendedZone", false)), precision)
	return &c
}

// ToRFC2822 is DateTime#toRFC2822(); nil is null.
func (dt *DateTime) ToRFC2822() *string {
	return toTechFormat(dt, "EEE, dd LLL yyyy HH:mm:ss ZZZ", false)
}

// ToHTTP is DateTime#toHTTP(); nil is null.
func (dt *DateTime) ToHTTP() *string {
	return toTechFormat(dt.ToUTC(), "EEE, dd LLL yyyy HH:mm:ss 'GMT'", true)
}

// ToSQLDate is DateTime#toSQLDate(); nil is null.
func (dt *DateTime) ToSQLDate() *string {
	if !dt.IsValid() {
		return nil
	}
	s := toISODate(dt, true, "")
	return &s
}

// ToSQLTime is DateTime#toSQLTime({includeOffset, includeZone, includeOffsetSpace}); nil is null.
func (dt *DateTime) ToSQLTime(opts ...any) *string {
	o := optObj(opts, 0)
	includeOffset := truthy(optDefault(o, "includeOffset", true))
	includeZone := truthy(optDefault(o, "includeZone", false))
	includeOffsetSpace := truthy(optDefault(o, "includeOffsetSpace", true))
	f := "HH:mm:ss.SSS"
	if includeZone || includeOffset {
		if includeOffsetSpace {
			f += " "
		}
		if includeZone {
			f += "z"
		} else if includeOffset {
			f += "ZZ"
		}
	}
	return toTechFormat(dt, f, true)
}

// ToSQL is DateTime#toSQL(opts); nil is null.
func (dt *DateTime) ToSQL(opts ...any) *string {
	if !dt.IsValid() {
		return nil
	}
	s := orNull(dt.ToSQLDate()) + " " + orNull(dt.ToSQLTime(opts...))
	return &s
}

// ToString is DateTime#toString().
func (dt *DateTime) ToString() string {
	if !dt.IsValid() {
		return invalidDateTimeString
	}
	return *dt.ToISO()
}

// ValueOf is DateTime#valueOf(): epoch milliseconds, NaN when invalid.
func (dt *DateTime) ValueOf() float64 { return dt.ToMillis() }

// ToMillis is DateTime#toMillis().
func (dt *DateTime) ToMillis() float64 {
	if !dt.IsValid() {
		return math.NaN()
	}
	return dt.ts
}

// ToSeconds is DateTime#toSeconds().
func (dt *DateTime) ToSeconds() float64 {
	if !dt.IsValid() {
		return math.NaN()
	}
	return dt.ts / 1000
}

// ToUnixInteger is DateTime#toUnixInteger().
func (dt *DateTime) ToUnixInteger() float64 {
	if !dt.IsValid() {
		return math.NaN()
	}
	return math.Floor(dt.ts / 1000)
}

// ToJSON is DateTime#toJSON(); nil is null.
func (dt *DateTime) ToJSON() *string { return dt.ToISO() }

// ToBSON is DateTime#toBSON().
func (dt *DateTime) ToBSON() JSDate { return dt.ToJSDate() }

// ToJSDate is DateTime#toJSDate().
func (dt *DateTime) ToJSDate() JSDate {
	if !dt.IsValid() {
		return JSDate(math.NaN())
	}
	return JSDate(timeClip(dt.ts))
}

// ToObject is DateTime#toObject({includeConfig}).
func (dt *DateTime) ToObject(opts ...any) Obj {
	if !dt.IsValid() {
		return Obj{}
	}
	c := dt.c
	base := Obj{"year", c.year, "month", c.month, "day", c.day, "hour", c.hour, "minute", c.minute,
		"second", c.second, "millisecond", c.millisecond}
	if truthy(optObj(opts, 0).get("includeConfig")) {
		base = append(base, "outputCalendar", ptrAny(dt.OutputCalendar()),
			"numberingSystem", ptrAny(dt.loc.numberingSystem), "locale", dt.loc.locale)
	}
	return base
}

// Diff is DateTime#diff(other, unit = "milliseconds", opts).
func (dt *DateTime) Diff(other *DateTime, args ...any) *Duration {
	if !dt.IsValid() || !other.IsValid() {
		return durationInvalid("created by diffing an invalid DateTime", nil)
	}
	unit := argAt(args, 0)
	if isUndef(unit) {
		unit = "milliseconds"
	}
	durOpts := objOf("locale", ptrAny(dt.Locale()), "numberingSystem", ptrAny(dt.NumberingSystem()))
	durOpts.assign(optObj(args, 1))
	var units []string
	if list := anyList(unit); list != nil {
		units = make([]string, len(list))
		for i, u := range list {
			units[i] = DurationNormalizeUnit(u)
		}
	} else {
		units = []string{DurationNormalizeUnit(unit)}
	}
	otherIsLater := other.ValueOf() > dt.ValueOf()
	earlier, later := other, dt
	if otherIsLater {
		earlier, later = dt, other
	}
	diffed := diffDateTimes(earlier, later, units, durOpts)
	if otherIsLater {
		return diffed.Negate()
	}
	return diffed
}

// DiffNow is DateTime#diffNow(unit, opts).
func (dt *DateTime) DiffNow(args ...any) *Duration { return dt.Diff(DateTimeNow(), args...) }

// Until is DateTime#until(other): an *Interval, or this DateTime when it is invalid.
func (dt *DateTime) Until(other any) any {
	if !dt.IsValid() {
		return dt
	}
	return IntervalFromDateTimes(dt, other)
}

// HasSame is DateTime#hasSame(other, unit, opts).
func (dt *DateTime) HasSame(other *DateTime, unit any, opts ...any) bool {
	if !dt.IsValid() {
		return false
	}
	inputMs := other.ValueOf()
	adjusted := dt.SetZone(other.zone, Obj{"keepLocalTime", true})
	return adjusted.StartOf(unit, opts...).ValueOf() <= inputMs && inputMs <= adjusted.EndOf(unit, opts...).ValueOf()
}

// Equals is DateTime#equals(other).
func (dt *DateTime) Equals(other *DateTime) bool {
	if !dt.IsValid() {
		return false
	}
	if other == nil {
		throw("TypeError", "Cannot read properties of null (reading 'isValid')")
	}
	return other.IsValid() && dt.ValueOf() == other.ValueOf() && dt.zone.Equals(other.zone) && dt.loc.equals(other.loc)
}

func relativeBase(o *jsObj, dt *DateTime) *DateTime {
	if b, ok := o.get("base").(*DateTime); ok && b != nil {
		return b
	}
	return dateTimeFromObject(newObj(), objOf("zone", dt.zone))
}

// ToRelative is DateTime#toRelative(options); nil is null.
func (dt *DateTime) ToRelative(options ...any) *string {
	if !dt.IsValid() {
		return nil
	}
	o := optObj(options, 0)
	base := relativeBase(o, dt)
	var padding any = 0.0
	if p := o.get("padding"); truthy(p) {
		padding = p
		if dt.ValueOf() < base.ValueOf() {
			padding = -toNumber(p)
		}
	}
	var units any = []any{"years", "months", "days", "hours", "minutes", "seconds"}
	unit := o.get("unit")
	if anyList(unit) != nil {
		units, unit = unit, Undefined
	}
	ro := o.clone()
	ro.set("numeric", "always")
	ro.set("units", units)
	ro.set("unit", unit)
	s := diffRelative(base, dt.Plus(padding), ro)
	return &s
}

// ToRelativeCalendar is DateTime#toRelativeCalendar(options); nil is null.
func (dt *DateTime) ToRelativeCalendar(options ...any) *string {
	if !dt.IsValid() {
		return nil
	}
	o := optObj(options, 0)
	base := relativeBase(o, dt)
	ro := o.clone()
	ro.set("numeric", "auto")
	ro.set("units", []any{"years", "months", "days"})
	ro.set("calendary", true)
	s := diffRelative(base, dt, ro)
	return &s
}
