package luxon

// impl/formats.js and impl/english.js

import "math"

const (
	fN = "numeric"
	fS = "short"
	fL = "long"
)

var (
	formatDateShort              = Obj{"year", fN, "month", fN, "day", fN}
	formatDateMed                = Obj{"year", fN, "month", fS, "day", fN}
	formatDateMedWithWeekday     = Obj{"year", fN, "month", fS, "day", fN, "weekday", fS}
	formatDateFull               = Obj{"year", fN, "month", fL, "day", fN}
	formatDateHuge               = Obj{"year", fN, "month", fL, "day", fN, "weekday", fL}
	formatTimeSimple             = Obj{"hour", fN, "minute", fN}
	formatTimeWithSeconds        = Obj{"hour", fN, "minute", fN, "second", fN}
	formatTimeWithShortOffset    = Obj{"hour", fN, "minute", fN, "second", fN, "timeZoneName", fS}
	formatTimeWithLongOffset     = Obj{"hour", fN, "minute", fN, "second", fN, "timeZoneName", fL}
	formatTime24Simple           = Obj{"hour", fN, "minute", fN, "hourCycle", "h23"}
	formatTime24WithSeconds      = Obj{"hour", fN, "minute", fN, "second", fN, "hourCycle", "h23"}
	formatTime24WithShortOffset  = Obj{"hour", fN, "minute", fN, "second", fN, "hourCycle", "h23", "timeZoneName", fS}
	formatTime24WithLongOffset   = Obj{"hour", fN, "minute", fN, "second", fN, "hourCycle", "h23", "timeZoneName", fL}
	formatDatetimeShort          = Obj{"year", fN, "month", fN, "day", fN, "hour", fN, "minute", fN}
	formatDatetimeShortWithSecs  = Obj{"year", fN, "month", fN, "day", fN, "hour", fN, "minute", fN, "second", fN}
	formatDatetimeMed            = Obj{"year", fN, "month", fS, "day", fN, "hour", fN, "minute", fN}
	formatDatetimeMedWithSecs    = Obj{"year", fN, "month", fS, "day", fN, "hour", fN, "minute", fN, "second", fN}
	formatDatetimeMedWithWeekday = Obj{"year", fN, "month", fS, "day", fN, "weekday", fS, "hour", fN, "minute", fN}
	formatDatetimeFull           = Obj{"year", fN, "month", fL, "day", fN, "hour", fN, "minute", fN, "timeZoneName", fS}
	formatDatetimeFullWithSecs   = Obj{"year", fN, "month", fL, "day", fN, "hour", fN, "minute", fN, "second", fN, "timeZoneName", fS}
	formatDatetimeHuge           = Obj{"year", fN, "month", fL, "day", fN, "weekday", fL, "hour", fN, "minute", fN, "timeZoneName", fL}
	formatDatetimeHugeWithSecs   = Obj{"year", fN, "month", fL, "day", fN, "weekday", fL, "hour", fN, "minute", fN, "second", fN, "timeZoneName", fL}
)

func cloneObj(o Obj) Obj { return append(Obj(nil), o...) }

// The format presets (DateTime.DATE_SHORT and so on). Each call returns a fresh Obj.

func DateTimeDATE_SHORT() Obj                  { return cloneObj(formatDateShort) }
func DateTimeDATE_MED() Obj                    { return cloneObj(formatDateMed) }
func DateTimeDATE_MED_WITH_WEEKDAY() Obj       { return cloneObj(formatDateMedWithWeekday) }
func DateTimeDATE_FULL() Obj                   { return cloneObj(formatDateFull) }
func DateTimeDATE_HUGE() Obj                   { return cloneObj(formatDateHuge) }
func DateTimeTIME_SIMPLE() Obj                 { return cloneObj(formatTimeSimple) }
func DateTimeTIME_WITH_SECONDS() Obj           { return cloneObj(formatTimeWithSeconds) }
func DateTimeTIME_WITH_SHORT_OFFSET() Obj      { return cloneObj(formatTimeWithShortOffset) }
func DateTimeTIME_WITH_LONG_OFFSET() Obj       { return cloneObj(formatTimeWithLongOffset) }
func DateTimeTIME_24_SIMPLE() Obj              { return cloneObj(formatTime24Simple) }
func DateTimeTIME_24_WITH_SECONDS() Obj        { return cloneObj(formatTime24WithSeconds) }
func DateTimeTIME_24_WITH_SHORT_OFFSET() Obj   { return cloneObj(formatTime24WithShortOffset) }
func DateTimeTIME_24_WITH_LONG_OFFSET() Obj    { return cloneObj(formatTime24WithLongOffset) }
func DateTimeDATETIME_SHORT() Obj              { return cloneObj(formatDatetimeShort) }
func DateTimeDATETIME_SHORT_WITH_SECONDS() Obj { return cloneObj(formatDatetimeShortWithSecs) }
func DateTimeDATETIME_MED() Obj                { return cloneObj(formatDatetimeMed) }
func DateTimeDATETIME_MED_WITH_SECONDS() Obj   { return cloneObj(formatDatetimeMedWithSecs) }
func DateTimeDATETIME_MED_WITH_WEEKDAY() Obj   { return cloneObj(formatDatetimeMedWithWeekday) }
func DateTimeDATETIME_FULL() Obj               { return cloneObj(formatDatetimeFull) }
func DateTimeDATETIME_FULL_WITH_SECONDS() Obj  { return cloneObj(formatDatetimeFullWithSecs) }
func DateTimeDATETIME_HUGE() Obj               { return cloneObj(formatDatetimeHuge) }
func DateTimeDATETIME_HUGE_WITH_SECONDS() Obj  { return cloneObj(formatDatetimeHugeWithSecs) }

// ---------------------------------------------------------------------------------------------
// English

var (
	monthsLong     = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	monthsShort    = []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	monthsNarrow   = []string{"J", "F", "M", "A", "M", "J", "J", "A", "S", "O", "N", "D"}
	weekdaysLong   = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	weekdaysShort  = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	weekdaysNarrow = []string{"M", "T", "W", "T", "F", "S", "S"}
	meridiemsEn    = []string{"AM", "PM"}
	erasLong       = []string{"Before Christ", "Anno Domini"}
	erasShort      = []string{"BC", "AD"}
	erasNarrow     = []string{"B", "A"}
)

func copyStrings(s []string) []string { return append([]string(nil), s...) }

// englishMonths returns nil where Luxon returns null.
func englishMonths(length any) []string {
	switch length {
	case "narrow":
		return copyStrings(monthsNarrow)
	case "short":
		return copyStrings(monthsShort)
	case "long":
		return copyStrings(monthsLong)
	case "numeric":
		return []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	case "2-digit":
		return []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12"}
	}
	return nil
}

func englishWeekdays(length any) []string {
	switch length {
	case "narrow":
		return copyStrings(weekdaysNarrow)
	case "short":
		return copyStrings(weekdaysShort)
	case "long":
		return copyStrings(weekdaysLong)
	case "numeric":
		return []string{"1", "2", "3", "4", "5", "6", "7"}
	}
	return nil
}

func englishEras(length any) []string {
	switch length {
	case "narrow":
		return copyStrings(erasNarrow)
	case "short":
		return copyStrings(erasShort)
	case "long":
		return copyStrings(erasLong)
	}
	return nil
}

// indexOrUndefined is arr[i] where i may fall outside the array (undefined, shown as nil).
func indexOrUndefined(arr []string, i float64) *string {
	if arr == nil {
		throw("TypeError", "Cannot read properties of null")
	}
	if !isIntegerF(i) || i < 0 || int(i) >= len(arr) {
		return nil
	}
	return &arr[int(i)]
}

func meridiemForDateTime(dt *DateTime) string {
	if dt.Hour() < 12 {
		return meridiemsEn[0]
	}
	return meridiemsEn[1]
}

func weekdayForDateTime(dt *DateTime, length string) *string {
	return indexOrUndefined(englishWeekdays(length), dt.Weekday()-1)
}

func monthForDateTime(dt *DateTime, length string) *string {
	return indexOrUndefined(englishMonths(length), dt.Month()-1)
}

func eraForDateTime(dt *DateTime, length string) *string {
	i := 1.0
	if dt.Year() < 0 {
		i = 0
	}
	return indexOrUndefined(englishEras(length), i)
}

// formatRelativeTime is English.formatRelativeTime.
func formatRelativeTime(unit string, count float64, numeric string, narrow bool) string {
	units := map[string][]string{
		"years":    {"year", "yr."},
		"quarters": {"quarter", "qtr."},
		"months":   {"month", "mo."},
		"weeks":    {"week", "wk."},
		"days":     {"day", "day", "days"},
		"hours":    {"hour", "hr."},
		"minutes":  {"minute", "min."},
		"seconds":  {"second", "sec."},
	}
	lastable := unit != "hours" && unit != "minutes" && unit != "seconds"
	if numeric == "auto" && lastable {
		isDay := unit == "days"
		name := func() string {
			u, ok := units[unit]
			if !ok {
				throw("TypeError", "Cannot read properties of undefined (reading '0')")
			}
			return u[0]
		}
		switch count {
		case 1:
			if isDay {
				return "tomorrow"
			}
			return "next " + name()
		case -1:
			if isDay {
				return "yesterday"
			}
			return "last " + name()
		case 0:
			if isDay {
				return "today"
			}
			return "this " + name()
		}
	}
	isInPast := isNegZero(count) || count < 0
	fmtValue := math.Abs(count)
	singular := fmtValue == 1
	lil, ok := units[unit]
	missing := func(index string) {
		if !ok {
			throw("TypeError", "Cannot read properties of undefined (reading '"+index+"')")
		}
	}
	var fmtUnit string
	switch {
	case narrow && singular:
		missing("1")
		fmtUnit = lil[1]
	case narrow:
		missing("2")
		if len(lil) > 2 {
			fmtUnit = lil[2]
		} else {
			fmtUnit = lil[1]
		}
	case singular:
		missing("0")
		fmtUnit = lil[0]
	default:
		fmtUnit = unit
	}
	if isInPast {
		return numberToString(fmtValue) + " " + fmtUnit + " ago"
	}
	return "in " + numberToString(fmtValue) + " " + fmtUnit
}
