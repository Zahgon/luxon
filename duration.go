package luxon

// duration.js

import (
	"math"
	"regexp"
	"strings"
)

const invalidDurationString = "Invalid Duration"

type durationMatrix map[string]map[string]float64

// conv is matrix[from][to]; a missing entry is undefined, which arithmetic turns into NaN.
func (m durationMatrix) conv(from, to string) float64 {
	if row, ok := m[from]; ok {
		if v, ok := row[to]; ok {
			return v
		}
	}
	return math.NaN()
}

var (
	daysInYearAccurate  = 146097.0 / 400
	daysInMonthAccurate = 146097.0 / 4800

	lowOrderMatrix = durationMatrix{
		"weeks": {"days": 7, "hours": 7 * 24, "minutes": 7 * 24 * 60, "seconds": 7 * 24 * 60 * 60,
			"milliseconds": 7 * 24 * 60 * 60 * 1000},
		"days":    {"hours": 24, "minutes": 24 * 60, "seconds": 24 * 60 * 60, "milliseconds": 24 * 60 * 60 * 1000},
		"hours":   {"minutes": 60, "seconds": 60 * 60, "milliseconds": 60 * 60 * 1000},
		"minutes": {"seconds": 60, "milliseconds": 60 * 1000},
		"seconds": {"milliseconds": 1000},
	}
	casualMatrix = withLowOrder(durationMatrix{
		"years": {"quarters": 4, "months": 12, "weeks": 52, "days": 365, "hours": 365 * 24,
			"minutes": 365 * 24 * 60, "seconds": 365 * 24 * 60 * 60, "milliseconds": 365 * 24 * 60 * 60 * 1000},
		"quarters": {"months": 3, "weeks": 13, "days": 91, "hours": 91 * 24, "minutes": 91 * 24 * 60,
			"seconds": 91 * 24 * 60 * 60, "milliseconds": 91 * 24 * 60 * 60 * 1000},
		"months": {"weeks": 4, "days": 30, "hours": 30 * 24, "minutes": 30 * 24 * 60,
			"seconds": 30 * 24 * 60 * 60, "milliseconds": 30 * 24 * 60 * 60 * 1000},
	})
	accurateMatrix = withLowOrder(durationMatrix{
		"years": {
			"quarters":     4,
			"months":       12,
			"weeks":        daysInYearAccurate / 7,
			"days":         daysInYearAccurate,
			"hours":        daysInYearAccurate * 24,
			"minutes":      daysInYearAccurate * 24 * 60,
			"seconds":      daysInYearAccurate * 24 * 60 * 60,
			"milliseconds": daysInYearAccurate * 24 * 60 * 60 * 1000,
		},
		"quarters": {
			"months":       3,
			"weeks":        daysInYearAccurate / 28,
			"days":         daysInYearAccurate / 4,
			"hours":        (daysInYearAccurate * 24) / 4,
			"minutes":      (daysInYearAccurate * 24 * 60) / 4,
			"seconds":      (daysInYearAccurate * 24 * 60 * 60) / 4,
			"milliseconds": (daysInYearAccurate * 24 * 60 * 60 * 1000) / 4,
		},
		"months": {
			"weeks":        daysInMonthAccurate / 7,
			"days":         daysInMonthAccurate,
			"hours":        daysInMonthAccurate * 24,
			"minutes":      daysInMonthAccurate * 24 * 60,
			"seconds":      daysInMonthAccurate * 24 * 60 * 60,
			"milliseconds": daysInMonthAccurate * 24 * 60 * 60 * 1000,
		},
	})

	orderedUnits = []string{"years", "quarters", "months", "weeks", "days", "hours", "minutes", "seconds", "milliseconds"}
	reverseUnits = []string{"milliseconds", "seconds", "minutes", "hours", "days", "weeks", "months", "quarters", "years"}
)

func withLowOrder(m durationMatrix) durationMatrix {
	for k, v := range lowOrderMatrix {
		m[k] = v
	}
	return m
}

// toMatrix reads the matrix option: an object of objects of numbers, or nil when not given.
func toMatrix(v any) durationMatrix {
	o, ok := v.(Obj)
	if !ok || !truthy(v) {
		return nil
	}
	m := durationMatrix{}
	outer := objFrom(o)
	for _, k := range outer.keys {
		row := map[string]float64{}
		if inner, ok := outer.vals[k].(Obj); ok {
			io := objFrom(inner)
			for _, k2 := range io.keys {
				row[k2] = toNumber(io.vals[k2])
			}
		}
		m[k] = row
	}
	return m
}

// Duration is a length of time: Luxon's Duration class.
type Duration struct {
	values             *jsObj
	loc                *locale
	conversionAccuracy string
	invalid            *invalid
	matrix             durationMatrix
}

func newDuration(values *jsObj, loc *locale, conversionAccuracy any, matrix durationMatrix, inv *invalid) *Duration {
	accurate := conversionAccuracy == "longterm"
	m := casualMatrix
	if accurate {
		m = accurateMatrix
	}
	if matrix != nil {
		m = matrix
	}
	if loc == nil {
		loc = createLocale(nil, nil, nil, nil, false)
	}
	ca := "casual"
	if accurate {
		ca = "longterm"
	}
	return &Duration{values: values, loc: loc, conversionAccuracy: ca, invalid: inv, matrix: m}
}

// durationAlts are clone's alternative settings; zero fields keep the original's.
type durationAlts struct {
	values             *jsObj
	loc                any
	conversionAccuracy any
	matrix             durationMatrix
}

func cloneDuration(dur *Duration, alts durationAlts, clear bool) *Duration {
	var values *jsObj
	if clear {
		values = alts.values
	} else {
		values = newObj()
		if dur.values != nil {
			values.assign(dur.values)
		}
		values.assign(alts.values)
	}
	ca := alts.conversionAccuracy
	if !truthy(ca) {
		ca = dur.conversionAccuracy
	}
	m := alts.matrix
	if m == nil {
		m = dur.matrix
	}
	return newDuration(values, dur.loc.clone(alts.loc), ca, m, nil)
}

func durationToMillis(matrix durationMatrix, vals *jsObj) float64 {
	sum := 0.0
	if v := vals.get("milliseconds"); !isNullish(v) {
		sum = fnum(v)
	}
	for _, unit := range reverseUnits[1:] {
		if v := vals.get(unit); truthy(v) {
			sum += float64(fnum(v) * matrix.conv(unit, "milliseconds"))
		}
	}
	return sum
}

func removeZeroes(vals *jsObj) *jsObj {
	out := newObj()
	for _, k := range vals.keys {
		if f, ok := vals.vals[k].(float64); !ok || f != 0 {
			out.set(k, vals.vals[k])
		}
	}
	return out
}

var reTrailingFractionZeros = regexp.MustCompile(`\.?0+$`)

func toISONumber(value float64) string {
	s := numberToString(value)
	if strings.Contains(s, "e") {
		return reTrailingFractionZeros.ReplaceAllString(toFixed(value, 20), "")
	}
	return s
}

// DurationFromMillis is Duration.fromMillis(count, opts).
func DurationFromMillis(count any, opts ...any) *Duration {
	return DurationFromObject(Obj{"milliseconds", count}, opts...)
}

// DurationFromObject is Duration.fromObject(obj, opts): obj is an Obj of units and values;
// opts may hold locale, numberingSystem, conversionAccuracy and matrix.
func DurationFromObject(obj any, opts ...any) *Duration {
	var src *jsObj
	switch o := obj.(type) {
	case Obj:
		src = objFrom(o)
	case *jsObj:
		src = o
	default:
		got := "null"
		if obj != nil {
			got = typeOf(obj)
		}
		throwInvalidArgument("Duration.fromObject: argument expected to be an object, got " + got)
	}
	o := optObj(opts, 0)
	values := normalizeObject(src, durationNormalizeUnit)
	return newDuration(values, newLocaleFromObject(o), o.get("conversionAccuracy"), toMatrix(o.get("matrix")), nil)
}

// DurationFromDurationLike is Duration.fromDurationLike: a number of milliseconds, a
// Duration, or an object.
func DurationFromDurationLike(durationLike any) *Duration {
	switch d := durationLike.(type) {
	case *Duration:
		return d
	case Obj, *jsObj, nil:
		return DurationFromObject(d)
	}
	if isNumber(durationLike) {
		return DurationFromMillis(durationLike)
	}
	throwInvalidArgument("Unknown duration argument " + toJSString(durationLike) + " of type " + typeOf(durationLike))
	return nil
}

// DurationFromISO is Duration.fromISO(text, opts).
func DurationFromISO(text string, opts ...any) *Duration {
	if parsed := parseISODuration(text); parsed != nil {
		return DurationFromObject(parsed, opts...)
	}
	return durationInvalid("unparsable", strp(`the input "`+text+`" can't be parsed as ISO 8601`))
}

// DurationFromISOTime is Duration.fromISOTime(text, opts).
func DurationFromISOTime(text string, opts ...any) *Duration {
	if parsed := parseISOTimeOnly(text); parsed != nil {
		return DurationFromObject(parsed, opts...)
	}
	return durationInvalid("unparsable", strp(`the input "`+text+`" can't be parsed as ISO 8601`))
}

// DurationInvalid is Duration.invalid(reason, explanation).
func DurationInvalid(reason string, explanation ...any) *Duration {
	return durationInvalid(reason, optString(argAt(explanation, 0)))
}

func durationInvalid(reason string, explanation *string) *Duration {
	if reason == "" {
		throwInvalidArgument("need to specify a reason the Duration is invalid")
	}
	inv := newInvalid(reason, explanation)
	if truthy(settingsThrowOnInvalid) {
		throwInvalidDuration(inv)
	}
	return newDuration(nil, nil, nil, nil, inv)
}

var durationUnitNames = map[string]string{
	"year": "years", "years": "years", "quarter": "quarters", "quarters": "quarters",
	"month": "months", "months": "months", "week": "weeks", "weeks": "weeks", "day": "days", "days": "days",
	"hour": "hours", "hours": "hours", "minute": "minutes", "minutes": "minutes",
	"second": "seconds", "seconds": "seconds", "millisecond": "milliseconds", "milliseconds": "milliseconds",
}

// DurationNormalizeUnit is Duration.normalizeUnit: "Hour" is "hours"; an unknown unit throws
// InvalidUnitError.
func DurationNormalizeUnit(unit any) string {
	if s, ok := unit.(string); ok {
		if n, ok := durationUnitNames[jsToLower(s)]; ok {
			return n
		}
	} else if truthy(unit) {
		throw("TypeError", "unit.toLowerCase is not a function")
	}
	throwInvalidUnit(unit)
	return ""
}

func durationNormalizeUnit(unit string) string { return DurationNormalizeUnit(unit) }

// DurationIsDuration is Duration.isDuration.
func DurationIsDuration(o any) bool {
	d, ok := o.(*Duration)
	return ok && d != nil
}

// Locale is the locale of a valid Duration, nil otherwise.
func (d *Duration) Locale() *string {
	if !d.IsValid() {
		return nil
	}
	return strp(d.loc.locale)
}

// NumberingSystem is the numbering system of a valid Duration, nil otherwise.
func (d *Duration) NumberingSystem() *string {
	if !d.IsValid() {
		return nil
	}
	return d.loc.numberingSystem
}

// ConversionAccuracy is "casual" or "longterm".
func (d *Duration) ConversionAccuracy() string { return d.conversionAccuracy }

// ToFormat is Duration#toFormat(fmt, opts).
func (d *Duration) ToFormat(fmt string, opts ...any) string {
	o := optObj(opts, 0)
	fmtOpts := o.clone()
	fmtOpts.set("floor", o.get("round") != false && o.get("floor") != false)
	if !d.IsValid() {
		return invalidDurationString
	}
	return newFormatter(d.loc, fmtOpts).formatDurationFromString(d, fmt)
}

// ToHuman is Duration#toHuman(opts).
func (d *Duration) ToHuman(opts ...any) string {
	if !d.IsValid() {
		return invalidDurationString
	}
	o := optObj(opts, 0)
	showZeros := o.get("showZeros") != false
	var l []string
	for _, unit := range orderedUnits {
		if unit == "quarters" {
			continue
		}
		val := d.values.get(unit)
		if unit == "months" {
			if val2 := d.values.get("quarters"); truthy(val2) {
				base := 0.0
				if !isNullish(val) {
					base = fnum(val)
				}
				val = base + float64(fnum(val2)*d.matrix.conv("quarters", "months"))
			}
		}
		if isUndef(val) {
			continue
		}
		if f, ok := val.(float64); ok && f == 0 && !showZeros {
			continue
		}
		nf := objOf("style", "unit", "unitDisplay", "long").assign(o)
		nf.set("unit", unit[:len(unit)-1])
		if s := d.loc.numberFormatter(nf).format(fnum(val)); s != "" {
			l = append(l, s)
		}
	}
	style := o.get("listStyle")
	if !truthy(style) {
		style = "narrow"
	}
	lf := objOf("type", "conjunction", "style", style).assign(o)
	return d.loc.listFormatter(lf).Format(l)
}

// ToObject is Duration#toObject.
func (d *Duration) ToObject() Obj {
	if !d.IsValid() {
		return Obj{}
	}
	return d.values.toObj()
}

// ToISO is Duration#toISO; nil for an invalid Duration.
func (d *Duration) ToISO() *string {
	if !d.IsValid() {
		return nil
	}
	s := "P"
	if d.Years() != 0 {
		s += toISONumber(d.Years()) + "Y"
	}
	if d.Months() != 0 || d.Quarters() != 0 {
		s += toISONumber(d.Months()+float64(d.Quarters()*3)) + "M"
	}
	if d.Weeks() != 0 {
		s += toISONumber(d.Weeks()) + "W"
	}
	if d.Days() != 0 {
		s += toISONumber(d.Days()) + "D"
	}
	if d.Hours() != 0 || d.Minutes() != 0 || d.Seconds() != 0 || d.Milliseconds() != 0 {
		s += "T"
	}
	if d.Hours() != 0 {
		s += toISONumber(d.Hours()) + "H"
	}
	if d.Minutes() != 0 {
		s += toISONumber(d.Minutes()) + "M"
	}
	if d.Seconds() != 0 || d.Milliseconds() != 0 {
		s += toISONumber(roundTo(d.Seconds()+d.Milliseconds()/1000, 3, "round")) + "S"
	}
	if s == "P" {
		s += "T0S"
	}
	return &s
}

// ToISOTime is Duration#toISOTime(opts); nil outside [0, 24h) or for an invalid Duration.
func (d *Duration) ToISOTime(opts ...any) *string {
	if !d.IsValid() {
		return nil
	}
	millis := d.ToMillis()
	if millis < 0 || millis >= 86400000 {
		return nil
	}
	o := objOf("suppressMilliseconds", false, "suppressSeconds", false, "includePrefix", false, "format", "extended")
	o.assign(optObj(opts, 0))
	o.set("includeOffset", false)
	return DateTimeFromMillis(millis, Obj{"zone", "UTC"}).ToISOTime(o.toObj())
}

// ToJSON is Duration#toJSON.
func (d *Duration) ToJSON() *string { return d.ToISO() }

// ToString is Duration#toString.
func (d *Duration) ToString() *string { return d.ToISO() }

// ToMillis is Duration#toMillis; NaN for an invalid Duration.
func (d *Duration) ToMillis() float64 {
	if !d.IsValid() {
		return math.NaN()
	}
	return durationToMillis(d.matrix, d.values)
}

// ValueOf is Duration#valueOf.
func (d *Duration) ValueOf() float64 { return d.ToMillis() }

// Plus is Duration#plus(duration): a Duration, an Obj, or a number of milliseconds.
func (d *Duration) Plus(duration any) *Duration {
	if !d.IsValid() {
		return d
	}
	dur := DurationFromDurationLike(duration)
	if dur.values == nil {
		throw("TypeError", "Cannot convert undefined or null to object")
	}
	result := newObj()
	for _, k := range orderedUnits {
		if dur.values.has(k) || d.values.has(k) {
			result.set(k, dur.Get(k)+d.Get(k))
		}
	}
	return cloneDuration(d, durationAlts{values: result}, true)
}

// Minus is Duration#minus(duration).
func (d *Duration) Minus(duration any) *Duration {
	if !d.IsValid() {
		return d
	}
	return d.Plus(DurationFromDurationLike(duration).Negate())
}

// MapUnits is Duration#mapUnits(fn): fn receives each value and its unit.
func (d *Duration) MapUnits(fn func(x float64, unit string) any) *Duration {
	if !d.IsValid() {
		return d
	}
	result := newObj()
	for _, k := range d.values.keys {
		result.set(k, asNumber(fn(fnum(d.values.vals[k]), k)))
	}
	return cloneDuration(d, durationAlts{values: result}, true)
}

// Get is Duration#get(unit).
func (d *Duration) Get(unit string) float64 { return d.unit(DurationNormalizeUnit(unit)) }

func (d *Duration) unit(u string) float64 {
	if !d.IsValid() {
		return math.NaN()
	}
	v := d.values.get(u)
	if !truthy(v) {
		return 0
	}
	return fnum(v)
}

// Set is Duration#set(values).
func (d *Duration) Set(values Obj) *Duration {
	if !d.IsValid() {
		return d
	}
	mixed := d.values.clone().assign(normalizeObject(objFrom(values), durationNormalizeUnit))
	return cloneDuration(d, durationAlts{values: mixed}, false)
}

// Reconfigure is Duration#reconfigure({locale, numberingSystem, conversionAccuracy, matrix}).
func (d *Duration) Reconfigure(opts ...any) *Duration {
	o := optObj(opts, 0)
	loc := d.loc.clone(objOf("locale", o.get("locale"), "numberingSystem", o.get("numberingSystem")))
	return cloneDuration(d, durationAlts{loc: loc, conversionAccuracy: o.get("conversionAccuracy"), matrix: toMatrix(o.get("matrix"))}, false)
}

// As is Duration#as(unit).
func (d *Duration) As(unit string) float64 {
	if !d.IsValid() {
		return math.NaN()
	}
	return d.ShiftTo(unit).Get(unit)
}

// Normalize is Duration#normalize.
func (d *Duration) Normalize() *Duration {
	if !d.IsValid() {
		return d
	}
	return d.ShiftTo()
}

// Rescale is Duration#rescale.
func (d *Duration) Rescale() *Duration {
	if !d.IsValid() {
		return d
	}
	return cloneDuration(d, durationAlts{values: removeZeroes(objFrom(d.ShiftToAll().ToObject()))}, true)
}

// ShiftTo is Duration#shiftTo(...units).
func (d *Duration) ShiftTo(units ...any) *Duration {
	if !d.IsValid() {
		return d
	}
	var us []string
	if len(units) == 0 {
		us = append(us, d.values.keys...)
	} else {
		for _, u := range units {
			us = append(us, DurationNormalizeUnit(u))
		}
	}
	m := d.matrix
	built := newObj()
	accumulated := d.values.clone()
	lastUnit := ""
	lastUnitTotal := 0.0
	for i, k := range orderedUnits {
		if !containsStr(us, k) {
			continue
		}
		if lastUnit != "" {
			lastUnitTotal *= m.conv(lastUnit, k)
		}
		lastUnit = k
		own := 0.0
		for _, ak := range append([]string(nil), accumulated.keys...) {
			av := fnum(accumulated.vals[ak])
			switch {
			case ak == k:
				own += av
			case i > indexOfStr(orderedUnits, ak):
				own += float64(m.conv(ak, k) * av)
				accumulated.vals[ak] = 0.0
			default:
				conv := m.conv(k, ak)
				toConvert := math.Trunc(av / conv)
				accumulated.vals[ak] = av - float64(toConvert*conv)
				own += toConvert
			}
		}
		own = snapFloatingPoint(own)
		accumulated.set(k, jsMod(own, 1))
		b := math.Trunc(own)
		built.set(k, b)
		lastUnitTotal += b
	}
	for _, key := range accumulated.keys {
		av := fnum(accumulated.vals[key])
		if av != 0 {
			toAdd := av
			if key != lastUnit {
				toAdd = av / m.conv(lastUnit, key)
			}
			built.set(lastUnit, fnum(built.get(lastUnit))+toAdd)
			lastUnitTotal += toAdd
		}
	}
	if overallSign := mathSign(lastUnitTotal); overallSign != 0 {
		for i, unit := range reverseUnits {
			if !built.has(unit) {
				continue
			}
			unitValue := fnum(built.vals[unit])
			unitSign := mathSign(unitValue)
			if unitSign == 0 || unitSign == overallSign {
				continue
			}
			for _, higherUnit := range reverseUnits[i+1:] {
				if built.has(higherUnit) {
					conv := m.conv(higherUnit, unit)
					toBorrow := math.Trunc((unitValue + float64((conv-1)*unitSign)) / conv)
					built.vals[higherUnit] = fnum(built.vals[higherUnit]) + toBorrow
					built.vals[unit] = fnum(built.vals[unit]) - float64(toBorrow*conv)
					break
				}
			}
		}
	}
	for i, unit := range orderedUnits {
		if unit == lastUnit || !built.has(unit) {
			continue
		}
		unitValue := fnum(built.vals[unit])
		if isIntegerF(unitValue) {
			continue
		}
		for _, smallerUnit := range orderedUnits[i+1:] {
			if built.has(smallerUnit) {
				conv := m.conv(unit, smallerUnit)
				unitFrac := jsMod(unitValue, 1)
				built.vals[unit] = math.Trunc(unitValue)
				built.vals[smallerUnit] = fnum(built.vals[smallerUnit]) + float64(unitFrac*conv)
				break
			}
		}
	}
	for i, unit := range reverseUnits {
		if !containsStr(us, unit) {
			continue
		}
		for _, ak := range built.keys {
			if i > indexOfStr(reverseUnits, ak) {
				conv := m.conv(unit, ak)
				av := fnum(built.vals[ak])
				toConvert := math.Trunc(av / conv)
				built.vals[unit] = fnum(built.vals[unit]) + toConvert
				built.vals[ak] = av - float64(toConvert*conv)
			}
		}
	}
	return cloneDuration(d, durationAlts{values: built}, true)
}

// ShiftToAll is Duration#shiftToAll.
func (d *Duration) ShiftToAll() *Duration {
	if !d.IsValid() {
		return d
	}
	return d.ShiftTo("years", "months", "weeks", "days", "hours", "minutes", "seconds", "milliseconds")
}

// Negate is Duration#negate.
func (d *Duration) Negate() *Duration {
	if !d.IsValid() {
		return d
	}
	negated := newObj()
	for _, k := range d.values.keys {
		v := fnum(d.values.vals[k])
		if v == 0 {
			negated.set(k, 0.0)
		} else {
			negated.set(k, -v)
		}
	}
	return cloneDuration(d, durationAlts{values: negated}, true)
}

// RemoveZeros is Duration#removeZeros.
func (d *Duration) RemoveZeros() *Duration {
	if !d.IsValid() {
		return d
	}
	return cloneDuration(d, durationAlts{values: removeZeroes(d.values)}, true)
}

func (d *Duration) Years() float64        { return d.unit("years") }
func (d *Duration) Quarters() float64     { return d.unit("quarters") }
func (d *Duration) Months() float64       { return d.unit("months") }
func (d *Duration) Weeks() float64        { return d.unit("weeks") }
func (d *Duration) Days() float64         { return d.unit("days") }
func (d *Duration) Hours() float64        { return d.unit("hours") }
func (d *Duration) Minutes() float64      { return d.unit("minutes") }
func (d *Duration) Seconds() float64      { return d.unit("seconds") }
func (d *Duration) Milliseconds() float64 { return d.unit("milliseconds") }

// IsValid is Duration#isValid.
func (d *Duration) IsValid() bool { return d.invalid == nil }

// InvalidReason is nil for a valid Duration.
func (d *Duration) InvalidReason() *string {
	if d.invalid == nil {
		return nil
	}
	return strp(d.invalid.reason)
}

// InvalidExplanation is nil for a valid Duration or one invalid without explanation.
func (d *Duration) InvalidExplanation() *string {
	if d.invalid == nil {
		return nil
	}
	return d.invalid.explanation
}

// Equals is Duration#equals(other).
func (d *Duration) Equals(other *Duration) bool {
	if !d.IsValid() || !other.IsValid() {
		return false
	}
	if !d.loc.equals(other.loc) {
		return false
	}
	eq := func(v1, v2 any) bool {
		zeroish := func(v any) bool {
			f, ok := v.(float64)
			return isUndef(v) || (ok && f == 0)
		}
		if zeroish(v1) {
			return zeroish(v2)
		}
		f1, ok1 := v1.(float64)
		f2, ok2 := v2.(float64)
		return ok1 && ok2 && f1 == f2
	}
	for _, u := range orderedUnits {
		if !eq(d.values.get(u), other.values.get(u)) {
			return false
		}
	}
	return true
}
