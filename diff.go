package luxon

// impl/diff.js

import "math"

func dayDiff(earlier, later *DateTime) float64 {
	utcDayStart := func(dt *DateTime) float64 {
		return dt.ToUTC(0.0, Obj{"keepLocalTime", true}).StartOf("day").ValueOf()
	}
	ms := utcDayStart(later) - utcDayStart(earlier)
	return math.Floor(DurationFromMillis(ms).As("days"))
}

type differ struct {
	unit string
	diff func(a, b *DateTime) float64
}

var differs = []differ{
	{"years", func(a, b *DateTime) float64 { return b.Year() - a.Year() }},
	{"quarters", func(a, b *DateTime) float64 { return b.Quarter() - a.Quarter() + float64((b.Year()-a.Year())*4) }},
	{"months", func(a, b *DateTime) float64 { return b.Month() - a.Month() + float64((b.Year()-a.Year())*12) }},
	{"weeks", func(a, b *DateTime) float64 {
		days := dayDiff(a, b)
		return (days - jsMod(days, 7)) / 7
	}},
	{"days", dayDiff},
}

func highOrderDiffs(cursor, later *DateTime, units []string) (*DateTime, *jsObj, *DateTime, string) {
	results := newObj()
	earlier := cursor
	lowestOrder := ""
	var highWater *DateTime
	for _, d := range differs {
		if indexOfStr(units, d.unit) < 0 {
			continue
		}
		lowestOrder = d.unit
		results.set(d.unit, d.diff(cursor, later))
		highWater = earlier.Plus(results.toObj())
		if highWater.ValueOf() > later.ValueOf() {
			results.set(d.unit, fnum(results.get(d.unit))-1)
			cursor = earlier.Plus(results.toObj())
			if cursor.ValueOf() > later.ValueOf() {
				highWater = cursor
				results.set(d.unit, fnum(results.get(d.unit))-1)
				cursor = earlier.Plus(results.toObj())
			}
		} else {
			cursor = highWater
		}
	}
	return cursor, results, highWater, lowestOrder
}

// diffDateTimes is impl/diff.js's default export.
func diffDateTimes(earlier, later *DateTime, units []string, opts *jsObj) *Duration {
	cursor, results, highWater, lowestOrder := highOrderDiffs(earlier, later, units)
	remainingMillis := later.ValueOf() - cursor.ValueOf()
	var lowerOrderUnits []any
	for _, u := range units {
		if containsStr([]string{"hours", "minutes", "seconds", "milliseconds"}, u) {
			lowerOrderUnits = append(lowerOrderUnits, u)
		}
	}
	if len(lowerOrderUnits) == 0 {
		if highWater != nil && highWater.ValueOf() < later.ValueOf() {
			highWater = cursor.Plus(Obj{lowestOrder, 1.0})
		}
		if highWater != cursor {
			key := lowestOrder
			if key == "" {
				key = "undefined"
			}
			base := 0.0
			if v := results.get(key); truthy(v) {
				base = fnum(v)
			}
			span := math.NaN()
			if highWater != nil {
				span = highWater.ValueOf() - cursor.ValueOf()
			}
			results.set(key, base+remainingMillis/span)
		}
	}
	duration := DurationFromObject(results, opts)
	if len(lowerOrderUnits) > 0 {
		return DurationFromMillis(remainingMillis, opts).ShiftTo(lowerOrderUnits...).Plus(duration)
	}
	return duration
}
