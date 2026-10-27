package luxon

// interval.js

import (
	"math"
	"sort"
	"strings"
)

const invalidIntervalString = "Invalid Interval"

// Interval is a half-open span of time between two DateTimes: Luxon's Interval class.
type Interval struct {
	s, e    *DateTime
	invalid *invalid
}

// dtValue is +dt, NaN where JavaScript reads undefined.
func dtValue(dt *DateTime) float64 {
	if dt == nil {
		return math.NaN()
	}
	return dt.ValueOf()
}

// dtArg is a DateTime field as a JavaScript value: undefined when absent.
func dtArg(dt *DateTime) any {
	if dt == nil {
		return Undefined
	}
	return dt
}

func isoOrNull(s *string) string { return orNull(s) }

func validateStartEnd(start, end *DateTime) *Interval {
	switch {
	case start == nil || !start.IsValid():
		return intervalInvalid("missing or invalid start", nil)
	case end == nil || !end.IsValid():
		return intervalInvalid("missing or invalid end", nil)
	case end.ValueOf() < start.ValueOf():
		return intervalInvalid("end before start", strp("The end of an interval must be after its start, but you had start="+
			isoOrNull(start.ToISO())+" and end="+isoOrNull(end.ToISO())))
	}
	return nil
}

// IntervalInvalid is Interval.invalid(reason, explanation).
func IntervalInvalid(reason string, explanation ...any) *Interval {
	return intervalInvalid(reason, optString(argAt(explanation, 0)))
}

func intervalInvalid(reason string, explanation *string) *Interval {
	if reason == "" {
		throwInvalidArgument("need to specify a reason the Interval is invalid")
	}
	inv := newInvalid(reason, explanation)
	if truthy(settingsThrowOnInvalid) {
		throwInvalidInterval(inv)
	}
	return &Interval{invalid: inv}
}

// IntervalFromDateTimes is Interval.fromDateTimes(start, end); each end is a DateTime, a JSDate
// or an Obj for DateTimeFromObject.
func IntervalFromDateTimes(start, end any) *Interval {
	builtStart, builtEnd := friendlyDateTime(start), friendlyDateTime(end)
	if err := validateStartEnd(builtStart, builtEnd); err != nil {
		return err
	}
	return &Interval{s: builtStart, e: builtEnd}
}

// IntervalAfter is Interval.after(start, duration).
func IntervalAfter(start, duration any) *Interval {
	dur := DurationFromDurationLike(duration)
	dt := friendlyDateTime(start)
	return IntervalFromDateTimes(dt, dt.Plus(dur))
}

// IntervalBefore is Interval.before(end, duration).
func IntervalBefore(end, duration any) *Interval {
	dur := DurationFromDurationLike(duration)
	dt := friendlyDateTime(end)
	return IntervalFromDateTimes(dt.Minus(dur), dt)
}

// catchJS runs fn and reports whether it returned without a JavaScript exception; any other
// panic (such as a gap in the ICU data) propagates.
func catchJS(fn func()) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, isJS := r.(*Error); !isJS {
				panic(r)
			}
			ok = false
		}
	}()
	fn()
	return true
}

// IntervalFromISO is Interval.fromISO(text, opts).
func IntervalFromISO(text string, opts ...any) *Interval {
	o := optObj(opts, 0)
	zone, setZone := o.get("zone"), o.get("setZone")
	restOpts := o.clone()
	restOpts.del("zone")
	restOpts.del("setZone")
	parts := strings.SplitN(text, "/", 3)
	if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
		s, e := parts[0], parts[1]
		var start, end *DateTime
		startIsValid := catchJS(func() {
			so := restOpts.clone()
			so.set("zone", zone)
			so.set("setZone", true)
			start = DateTimeFromISO(s, so.toObj())
		}) && start.IsValid()
		endIsValid := catchJS(func() {
			vals, parsedZone := parseISOIntervalEnd(e)
			eo := restOpts.clone()
			if startIsValid {
				eo.set("overrideNow", start.ValueOf())
				eo.set("zone", start.Zone())
			} else {
				eo.set("overrideNow", nil)
				eo.set("zone", zone)
			}
			eo.set("setZone", true)
			end = parseDataToDateTime(vals, parsedZone, eo, "ISO 8601 Interval end", e, Undefined)
		}) && end.IsValid()
		if startIsValid && !truthy(setZone) {
			start = start.SetZone(zone)
		}
		if endIsValid && !truthy(setZone) {
			end = end.SetZone(zone)
		}
		if startIsValid && endIsValid {
			return IntervalFromDateTimes(start, end)
		}
		if startIsValid {
			if dur := DurationFromISO(e, opts...); dur.IsValid() {
				return IntervalAfter(start, dur)
			}
		} else if endIsValid {
			if dur := DurationFromISO(s, opts...); dur.IsValid() {
				return IntervalBefore(end, dur)
			}
		}
	}
	return intervalInvalid("unparsable", strp(`the input "`+text+`" can't be parsed as ISO 8601`))
}

// IntervalIsInterval is Interval.isInterval.
func IntervalIsInterval(o any) bool {
	i, ok := o.(*Interval)
	return ok && i != nil
}

// Start is nil for an invalid Interval.
func (i *Interval) Start() *DateTime {
	if !i.IsValid() {
		return nil
	}
	return i.s
}

// End is nil for an invalid Interval.
func (i *Interval) End() *DateTime {
	if !i.IsValid() {
		return nil
	}
	return i.e
}

// LastDateTime is the last millisecond inside the Interval.
func (i *Interval) LastDateTime() *DateTime {
	if !i.IsValid() || i.e == nil {
		return nil
	}
	return i.e.Minus(1.0)
}

func (i *Interval) IsValid() bool { return i.invalid == nil }

// InvalidReason is nil for a valid Interval.
func (i *Interval) InvalidReason() *string {
	if i.invalid == nil {
		return nil
	}
	return strp(i.invalid.reason)
}

func (i *Interval) InvalidExplanation() *string {
	if i.invalid == nil {
		return nil
	}
	return i.invalid.explanation
}

func unitOr(args []any, i int, def string) any {
	if v := argAt(args, i); !isUndef(v) {
		return v
	}
	return def
}

// Length is Interval#length(unit = "milliseconds").
func (i *Interval) Length(unit ...any) float64 {
	if !i.IsValid() {
		return math.NaN()
	}
	u := unitOr(unit, 0, "milliseconds")
	return i.ToDuration(u).Get(toJSString(u))
}

// Count is Interval#count(unit = "milliseconds", opts).
func (i *Interval) Count(args ...any) float64 {
	if !i.IsValid() {
		return math.NaN()
	}
	unit := toJSString(unitOr(args, 0, "milliseconds"))
	opts := argAt(args, 1)
	start := i.Start().StartOf(unit, opts)
	end := i.End()
	if o, ok := opts.(Obj); ok && truthy(opt(o, "useLocaleWeeks")) {
		end = end.Reconfigure(Obj{"locale", ptrAny(start.Locale())})
	}
	end = end.StartOf(unit, opts)
	n := math.Floor(end.Diff(start, unit).Get(unit))
	if end.ValueOf() != i.End().ValueOf() {
		n++
	}
	return n
}

// HasSame is Interval#hasSame(unit).
func (i *Interval) HasSame(unit string) bool {
	if !i.IsValid() {
		return false
	}
	if i.IsEmpty() {
		return i.s.HasSame(i.e, unit)
	}
	return i.e.Minus(1.0).HasSame(i.s, unit)
}

// IsEmpty is Interval#isEmpty.
func (i *Interval) IsEmpty() bool {
	if i.s == nil || i.e == nil {
		throw("TypeError", "Cannot read properties of undefined (reading 'valueOf')")
	}
	return i.s.ValueOf() == i.e.ValueOf()
}

func (i *Interval) IsAfter(dateTime *DateTime) bool {
	return i.IsValid() && dtValue(i.s) > dtValue(dateTime)
}

func (i *Interval) IsBefore(dateTime *DateTime) bool {
	return i.IsValid() && dtValue(i.e) <= dtValue(dateTime)
}

func (i *Interval) Contains(dateTime *DateTime) bool {
	return i.IsValid() && dtValue(i.s) <= dtValue(dateTime) && dtValue(i.e) > dtValue(dateTime)
}

// Set is Interval#set({start, end}).
func (i *Interval) Set(values ...any) *Interval {
	if !i.IsValid() {
		return i
	}
	o := optObj(values, 0)
	start, end := o.get("start"), o.get("end")
	if !truthy(start) {
		start = i.s
	}
	if !truthy(end) {
		end = i.e
	}
	return IntervalFromDateTimes(start, end)
}

// SplitAt is Interval#splitAt(...dateTimes).
func (i *Interval) SplitAt(dateTimes ...any) []*Interval {
	if !i.IsValid() {
		return []*Interval{}
	}
	var sorted []*DateTime
	for _, d := range dateTimes {
		if dt := friendlyDateTime(d); i.Contains(dt) {
			sorted = append(sorted, dt)
		}
	}
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].ToMillis()-sorted[b].ToMillis() < 0 })
	results := []*Interval{}
	s := i.s
	for idx := 0; s.ValueOf() < i.e.ValueOf(); idx++ {
		added := i.e
		if idx < len(sorted) {
			added = sorted[idx]
		}
		next := added
		if added.ValueOf() > i.e.ValueOf() {
			next = i.e
		}
		results = append(results, IntervalFromDateTimes(s, next))
		s = next
	}
	return results
}

// SplitBy is Interval#splitBy(duration).
func (i *Interval) SplitBy(duration any) []*Interval {
	dur := DurationFromDurationLike(duration)
	if !i.IsValid() || !dur.IsValid() || dur.As("milliseconds") == 0 {
		return []*Interval{}
	}
	results := []*Interval{}
	s := i.s
	for idx := 1.0; s.ValueOf() < i.e.ValueOf(); idx++ {
		k := idx
		added := i.Start().Plus(dur.MapUnits(func(x float64, _ string) any { return x * k }))
		next := added
		if added.ValueOf() > i.e.ValueOf() {
			next = i.e
		}
		results = append(results, IntervalFromDateTimes(s, next))
		s = next
	}
	return results
}

// DivideEqually is Interval#divideEqually(numberOfParts).
func (i *Interval) DivideEqually(numberOfParts float64) []*Interval {
	if !i.IsValid() {
		return []*Interval{}
	}
	parts := i.SplitBy(i.Length() / numberOfParts)
	n := float64(len(parts))
	end := toIntegerOrInfinity(numberOfParts)
	if end < 0 {
		end = math.Max(n+end, 0)
	} else {
		end = math.Min(end, n)
	}
	return parts[:int(end)]
}

func (i *Interval) Overlaps(other *Interval) bool {
	return dtValue(i.e) > dtValue(other.s) && dtValue(i.s) < dtValue(other.e)
}

func (i *Interval) AbutsStart(other *Interval) bool {
	return i.IsValid() && dtValue(i.e) == dtValue(other.s)
}

func (i *Interval) AbutsEnd(other *Interval) bool {
	return i.IsValid() && dtValue(other.e) == dtValue(i.s)
}

func (i *Interval) Engulfs(other *Interval) bool {
	return i.IsValid() && dtValue(i.s) <= dtValue(other.s) && dtValue(i.e) >= dtValue(other.e)
}

func (i *Interval) Equals(other *Interval) bool {
	if !i.IsValid() || !other.IsValid() {
		return false
	}
	return i.s.Equals(other.s) && i.e.Equals(other.e)
}

// Intersection is Interval#intersection(other); nil where Luxon returns null.
func (i *Interval) Intersection(other *Interval) *Interval {
	if !i.IsValid() {
		return i
	}
	s := other.s
	if dtValue(i.s) > dtValue(other.s) {
		s = i.s
	}
	e := other.e
	if dtValue(i.e) < dtValue(other.e) {
		e = i.e
	}
	if dtValue(s) >= dtValue(e) {
		return nil
	}
	return IntervalFromDateTimes(dtArg(s), dtArg(e))
}

func (i *Interval) Union(other *Interval) *Interval {
	if !i.IsValid() {
		return i
	}
	s := other.s
	if dtValue(i.s) < dtValue(other.s) {
		s = i.s
	}
	e := other.e
	if dtValue(i.e) > dtValue(other.e) {
		e = i.e
	}
	return IntervalFromDateTimes(dtArg(s), dtArg(e))
}

// IntervalMerge is Interval.merge(intervals); like the original it sorts intervals in place.
func IntervalMerge(intervals []*Interval) []*Interval {
	sort.SliceStable(intervals, func(a, b int) bool { return dtValue(intervals[a].s)-dtValue(intervals[b].s) < 0 })
	found := []*Interval{}
	var current *Interval
	for _, item := range intervals {
		switch {
		case current == nil:
			current = item
		case current.Overlaps(item) || current.AbutsStart(item):
			current = current.Union(item)
		default:
			found = append(found, current)
			current = item
		}
	}
	if current != nil {
		found = append(found, current)
	}
	return found
}

// IntervalXor is Interval.xor(intervals).
func IntervalXor(intervals []*Interval) []*Interval {
	type end struct {
		time  *DateTime
		start bool
	}
	var arr []end
	for _, i := range intervals {
		arr = append(arr, end{i.s, true}, end{i.e, false})
	}
	sort.SliceStable(arr, func(a, b int) bool { return dtValue(arr[a].time)-dtValue(arr[b].time) < 0 })
	results := []*Interval{}
	var start *DateTime
	currentCount := 0
	for _, x := range arr {
		if x.start {
			currentCount++
		} else {
			currentCount--
		}
		if currentCount == 1 {
			start = x.time
		} else {
			if start != nil && start.ValueOf() != dtValue(x.time) {
				results = append(results, IntervalFromDateTimes(start, dtArg(x.time)))
			}
			start = nil
		}
	}
	return IntervalMerge(results)
}

// Difference is Interval#difference(...intervals).
func (i *Interval) Difference(intervals ...*Interval) []*Interval {
	out := []*Interval{}
	for _, x := range IntervalXor(append([]*Interval{i}, intervals...)) {
		if r := i.Intersection(x); r != nil && !r.IsEmpty() {
			out = append(out, r)
		}
	}
	return out
}

// ToString is Interval#toString.
func (i *Interval) ToString() string {
	if !i.IsValid() {
		return invalidIntervalString
	}
	return "[" + orNull(i.s.ToISO()) + " – " + orNull(i.e.ToISO()) + ")"
}

// ToLocaleString is Interval#toLocaleString(formatOpts = DATE_SHORT, opts).
func (i *Interval) ToLocaleString(args ...any) string {
	if !i.IsValid() {
		return invalidIntervalString
	}
	formatOpts := formatDateShort
	if o, ok := argAt(args, 0).(Obj); ok {
		formatOpts = o
	}
	return newFormatter(i.s.loc.clone(optObj(args, 1)), objFrom(formatOpts)).formatInterval(i)
}

// ToISO is Interval#toISO(opts).
func (i *Interval) ToISO(opts ...any) string {
	if !i.IsValid() {
		return invalidIntervalString
	}
	return orNull(i.s.ToISO(opts...)) + "/" + orNull(i.e.ToISO(opts...))
}

func (i *Interval) ToISODate() string {
	if !i.IsValid() {
		return invalidIntervalString
	}
	return orNull(i.s.ToISODate()) + "/" + orNull(i.e.ToISODate())
}

func (i *Interval) ToISOTime(opts ...any) string {
	if !i.IsValid() {
		return invalidIntervalString
	}
	return orNull(i.s.ToISOTime(opts...)) + "/" + orNull(i.e.ToISOTime(opts...))
}

// ToFormat is Interval#toFormat(dateFormat, {separator = " – "}).
func (i *Interval) ToFormat(dateFormat string, opts ...any) string {
	if !i.IsValid() {
		return invalidIntervalString
	}
	sep := optObj(opts, 0).get("separator")
	if isUndef(sep) {
		sep = " – "
	}
	return i.s.ToFormat(dateFormat) + toJSString(sep) + i.e.ToFormat(dateFormat)
}

// ToDuration is Interval#toDuration(unit, opts).
func (i *Interval) ToDuration(args ...any) *Duration {
	if !i.IsValid() {
		return durationInvalid(*i.InvalidReason(), nil)
	}
	return i.e.Diff(i.s, args...)
}

// MapEndpoints is Interval#mapEndpoints(mapFn).
func (i *Interval) MapEndpoints(mapFn func(*DateTime) any) *Interval {
	return IntervalFromDateTimes(mapFn(i.s), mapFn(i.e))
}
