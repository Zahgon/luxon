package luxon

import "math"

// Get is DateTime#get(unit), this[unit]: the getter of that name, or undefined.
func (dt *DateTime) Get(unit string) any {
	switch unit {
	case "year":
		return dt.Year()
	case "quarter":
		return dt.Quarter()
	case "month":
		return dt.Month()
	case "day":
		return dt.Day()
	case "hour":
		return dt.Hour()
	case "minute":
		return dt.Minute()
	case "second":
		return dt.Second()
	case "millisecond":
		return dt.Millisecond()
	case "weekYear":
		return dt.WeekYear()
	case "weekNumber":
		return dt.WeekNumber()
	case "weekday":
		return dt.Weekday()
	case "localWeekday":
		return dt.LocalWeekday()
	case "localWeekNumber":
		return dt.LocalWeekNumber()
	case "localWeekYear":
		return dt.LocalWeekYear()
	case "ordinal":
		return dt.Ordinal()
	case "offset":
		return dt.Offset()
	case "daysInMonth":
		return dt.DaysInMonth()
	case "daysInYear":
		return dt.DaysInYear()
	case "weeksInWeekYear":
		return dt.WeeksInWeekYear()
	case "weeksInLocalWeekYear":
		return dt.WeeksInLocalWeekYear()
	case "isValid":
		return dt.IsValid()
	case "isWeekend":
		return dt.IsWeekend()
	case "isInDST":
		return dt.IsInDST()
	case "isInLeapYear":
		return dt.IsInLeapYear()
	case "wasHole":
		return dt.WasHole()
	case "isOffsetFixed":
		if b := dt.IsOffsetFixed(); b != nil {
			return *b
		}
		return nil
	case "invalidReason":
		return ptrAny(dt.InvalidReason())
	case "invalidExplanation":
		return ptrAny(dt.InvalidExplanation())
	case "locale":
		return ptrAny(dt.Locale())
	case "numberingSystem":
		return ptrAny(dt.NumberingSystem())
	case "outputCalendar":
		return ptrAny(dt.OutputCalendar())
	case "zoneName":
		return ptrAny(dt.ZoneName())
	case "monthShort":
		return ptrAny(dt.MonthShort())
	case "monthLong":
		return ptrAny(dt.MonthLong())
	case "weekdayShort":
		return ptrAny(dt.WeekdayShort())
	case "weekdayLong":
		return ptrAny(dt.WeekdayLong())
	case "offsetNameShort":
		return ptrAny(dt.OffsetNameShort())
	case "offsetNameLong":
		return ptrAny(dt.OffsetNameLong())
	case "zone":
		return dt.zone
	case "ts":
		return dt.ts
	case "isLuxonDateTime":
		return true
	}
	return Undefined
}

// IsValid is DateTime#isValid.
func (dt *DateTime) IsValid() bool { return dt.invalid == nil }

// InvalidReason is DateTime#invalidReason; nil is null.
func (dt *DateTime) InvalidReason() *string {
	if dt.invalid == nil {
		return nil
	}
	r := dt.invalid.reason
	return &r
}

// InvalidExplanation is DateTime#invalidExplanation; nil is null.
func (dt *DateTime) InvalidExplanation() *string {
	if dt.invalid == nil {
		return nil
	}
	return dt.invalid.explanation
}

// Locale is DateTime#locale; nil is null.
func (dt *DateTime) Locale() *string {
	if !dt.IsValid() {
		return nil
	}
	s := dt.loc.locale
	return &s
}

// NumberingSystem is DateTime#numberingSystem; nil is null.
func (dt *DateTime) NumberingSystem() *string {
	if !dt.IsValid() {
		return nil
	}
	return dt.loc.numberingSystem
}

// OutputCalendar is DateTime#outputCalendar; nil is null.
func (dt *DateTime) OutputCalendar() *string {
	if !dt.IsValid() {
		return nil
	}
	return dt.loc.outputCalendar
}

// Zone is DateTime#zone.
func (dt *DateTime) Zone() Zone { return dt.zone }

// ZoneName is DateTime#zoneName; nil is null.
func (dt *DateTime) ZoneName() *string {
	if !dt.IsValid() {
		return nil
	}
	s := dt.zone.Name()
	return &s
}

// WasHole is DateTime#wasHole: the requested local time fell in a gap and was shifted.
func (dt *DateTime) WasHole() bool { return dt.wasHole }

func (dt *DateTime) cal(v func(c *calObj) float64) float64 {
	if !dt.IsValid() {
		return math.NaN()
	}
	return v(dt.c)
}

// Year is DateTime#year.
func (dt *DateTime) Year() float64 { return dt.cal(func(c *calObj) float64 { return c.year }) }

// Quarter is DateTime#quarter.
func (dt *DateTime) Quarter() float64 {
	return dt.cal(func(c *calObj) float64 { return math.Ceil(c.month / 3) })
}

// Month is DateTime#month.
func (dt *DateTime) Month() float64 { return dt.cal(func(c *calObj) float64 { return c.month }) }

// Day is DateTime#day.
func (dt *DateTime) Day() float64 { return dt.cal(func(c *calObj) float64 { return c.day }) }

// Hour is DateTime#hour.
func (dt *DateTime) Hour() float64 { return dt.cal(func(c *calObj) float64 { return c.hour }) }

// Minute is DateTime#minute.
func (dt *DateTime) Minute() float64 { return dt.cal(func(c *calObj) float64 { return c.minute }) }

// Second is DateTime#second.
func (dt *DateTime) Second() float64 { return dt.cal(func(c *calObj) float64 { return c.second }) }

// Millisecond is DateTime#millisecond.
func (dt *DateTime) Millisecond() float64 {
	return dt.cal(func(c *calObj) float64 { return c.millisecond })
}

func (dt *DateTime) isoWeekData() *weekObj {
	if dt.weekData == nil {
		w := gregorianToWeek(*dt.c, 4, 1)
		dt.weekData = &w
	}
	return dt.weekData
}

func (dt *DateTime) localWeek() *weekObj {
	if dt.localWeekData == nil {
		w := gregorianToWeek(*dt.c, dt.loc.getMinDaysInFirstWeek(), dt.loc.getStartOfWeek())
		dt.localWeekData = &w
	}
	return dt.localWeekData
}

// WeekYear is DateTime#weekYear.
func (dt *DateTime) WeekYear() float64 {
	return dt.cal(func(*calObj) float64 { return dt.isoWeekData().weekYear })
}

// WeekNumber is DateTime#weekNumber.
func (dt *DateTime) WeekNumber() float64 {
	return dt.cal(func(*calObj) float64 { return dt.isoWeekData().weekNumber })
}

// Weekday is DateTime#weekday (ISO: Monday is 1).
func (dt *DateTime) Weekday() float64 {
	return dt.cal(func(*calObj) float64 { return dt.isoWeekData().weekday })
}

// IsWeekend is DateTime#isWeekend.
func (dt *DateTime) IsWeekend() bool {
	if !dt.IsValid() {
		return false
	}
	wd := dt.Weekday()
	for _, d := range dt.loc.getWeekendDays() {
		if d == wd {
			return true
		}
	}
	return false
}

// LocalWeekday is DateTime#localWeekday.
func (dt *DateTime) LocalWeekday() float64 {
	return dt.cal(func(*calObj) float64 { return dt.localWeek().weekday })
}

// LocalWeekNumber is DateTime#localWeekNumber.
func (dt *DateTime) LocalWeekNumber() float64 {
	return dt.cal(func(*calObj) float64 { return dt.localWeek().weekNumber })
}

// LocalWeekYear is DateTime#localWeekYear.
func (dt *DateTime) LocalWeekYear() float64 {
	return dt.cal(func(*calObj) float64 { return dt.localWeek().weekYear })
}

// Ordinal is DateTime#ordinal.
func (dt *DateTime) Ordinal() float64 {
	return dt.cal(func(c *calObj) float64 { return gregorianToOrdinal(*c).ordinal })
}

// nthName is list[n - 1] of a month or weekday list; nil is null or undefined.
func nthName(list []any, n float64) *string {
	i := int(n) - 1
	if i < 0 || i >= len(list) {
		return nil
	}
	if s, ok := list[i].(string); ok {
		return &s
	}
	return nil
}

// MonthShort is DateTime#monthShort; nil is null.
func (dt *DateTime) MonthShort() *string {
	if !dt.IsValid() {
		return nil
	}
	return nthName(dt.loc.months("short", false), dt.c.month)
}

// MonthLong is DateTime#monthLong; nil is null.
func (dt *DateTime) MonthLong() *string {
	if !dt.IsValid() {
		return nil
	}
	return nthName(dt.loc.months("long", false), dt.c.month)
}

// WeekdayShort is DateTime#weekdayShort; nil is null.
func (dt *DateTime) WeekdayShort() *string {
	if !dt.IsValid() {
		return nil
	}
	return nthName(dt.loc.weekdays("short", false), dt.Weekday())
}

// WeekdayLong is DateTime#weekdayLong; nil is null.
func (dt *DateTime) WeekdayLong() *string {
	if !dt.IsValid() {
		return nil
	}
	return nthName(dt.loc.weekdays("long", false), dt.Weekday())
}

// Offset is DateTime#offset, in minutes.
func (dt *DateTime) Offset() float64 {
	if !dt.IsValid() {
		return math.NaN()
	}
	return dt.o
}

// OffsetNameShort is DateTime#offsetNameShort; nil is null.
func (dt *DateTime) OffsetNameShort() *string {
	if !dt.IsValid() {
		return nil
	}
	return dt.zone.OffsetName(dt.ts, Obj{"format", "short", "locale", ptrAny(dt.Locale())})
}

// OffsetNameLong is DateTime#offsetNameLong; nil is null.
func (dt *DateTime) OffsetNameLong() *string {
	if !dt.IsValid() {
		return nil
	}
	return dt.zone.OffsetName(dt.ts, Obj{"format", "long", "locale", ptrAny(dt.Locale())})
}

// IsOffsetFixed is DateTime#isOffsetFixed; nil is null.
func (dt *DateTime) IsOffsetFixed() *bool {
	if !dt.IsValid() {
		return nil
	}
	b := dt.zone.IsUniversal()
	return &b
}

// IsInDST is DateTime#isInDST.
func (dt *DateTime) IsInDST() bool {
	if derefBool(dt.IsOffsetFixed()) {
		return false
	}
	return dt.Offset() > dt.Set(Obj{"month", 1.0, "day", 1.0}).Offset() ||
		dt.Offset() > dt.Set(Obj{"month", 5.0}).Offset()
}

// GetPossibleOffsets is DateTime#getPossibleOffsets(): both readings of an ambiguous local time.
func (dt *DateTime) GetPossibleOffsets() []*DateTime {
	if !dt.IsValid() || derefBool(dt.IsOffsetFixed()) {
		return []*DateTime{dt}
	}
	const dayMs, minuteMs = 86400000.0, 60000.0
	localTS := objToLocalTS(*dt.c)
	oEarlier := dt.zone.Offset(localTS - dayMs)
	oLater := dt.zone.Offset(localTS + dayMs)
	o1 := dt.zone.Offset(localTS - float64(oEarlier*minuteMs))
	o2 := dt.zone.Offset(localTS - float64(oLater*minuteMs))
	if o1 == o2 {
		return []*DateTime{dt}
	}
	ts1 := localTS - float64(o1*minuteMs)
	ts2 := localTS - float64(o2*minuteMs)
	c1, c2 := tsToObj(ts1, o1), tsToObj(ts2, o2)
	if c1.hour == c2.hour && c1.minute == c2.minute && c1.second == c2.second && c1.millisecond == c2.millisecond {
		return []*DateTime{dt.clone(dtAlts{ts: &ts1}), dt.clone(dtAlts{ts: &ts2})}
	}
	return []*DateTime{dt}
}

// IsInLeapYear is DateTime#isInLeapYear.
func (dt *DateTime) IsInLeapYear() bool { return isLeapYear(dt.Year()) }

// DaysInMonth is DateTime#daysInMonth (NaN where the original gives undefined).
func (dt *DateTime) DaysInMonth() float64 { return daysInMonth(dt.Year(), dt.Month()) }

// DaysInYear is DateTime#daysInYear.
func (dt *DateTime) DaysInYear() float64 {
	return dt.cal(func(c *calObj) float64 { return daysInYear(c.year) })
}

// WeeksInWeekYear is DateTime#weeksInWeekYear.
func (dt *DateTime) WeeksInWeekYear() float64 {
	return dt.cal(func(*calObj) float64 { return weeksInWeekYear(dt.WeekYear(), 4, 1) })
}

// WeeksInLocalWeekYear is DateTime#weeksInLocalWeekYear.
func (dt *DateTime) WeeksInLocalWeekYear() float64 {
	return dt.cal(func(*calObj) float64 {
		return weeksInWeekYear(dt.LocalWeekYear(), dt.loc.getMinDaysInFirstWeek(), dt.loc.getStartOfWeek())
	})
}
