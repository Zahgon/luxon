package luxon

// InfoType carries the static methods of luxon's Info class.
type InfoType struct{}

// Info is luxon's Info class.
var Info InfoType

// nullDefault is a destructuring default of null.
func nullDefault(v any) any {
	if isUndef(v) {
		return nil
	}
	return v
}

// stringList reads a list of names; nil is null, and so is a nil element.
func stringList(l []any) []*string {
	if l == nil {
		return nil
	}
	out := make([]*string, len(l))
	for i, v := range l {
		if s, ok := v.(string); ok {
			out[i] = &s
		}
	}
	return out
}

// HasDST is Info.hasDST(zone = Settings.defaultZone).
func (InfoType) HasDST(args ...any) bool {
	zone := argAt(args, 0)
	if isUndef(zone) {
		zone = Settings.DefaultZone()
	}
	proto := DateTimeNow().SetZone(zone).Set(Obj{"month", 12.0})
	if zone == nil {
		throw("TypeError", "Cannot read properties of null (reading 'isUniversal')")
	}
	universal := false
	if z, ok := zone.(Zone); ok {
		universal = z.IsUniversal()
	}
	return !universal && proto.Offset() != proto.Set(Obj{"month", 6.0}).Offset()
}

// IsValidIANAZone is Info.isValidIANAZone(zone).
func (InfoType) IsValidIANAZone(zone any) bool { return IANAZoneIsValidZone(zone) }

// NormalizeZone is Info.normalizeZone(input).
func (InfoType) NormalizeZone(input any) Zone { return normalizeZone(input, Settings.DefaultZone()) }

func infoLocale(opts []any) *locale {
	return createLocale(nullDefault(optObj(opts, 0).get("locale")), nil, nil, nil, false)
}

// GetStartOfWeek is Info.getStartOfWeek({locale}).
func (InfoType) GetStartOfWeek(opts ...any) float64 { return infoLocale(opts).getStartOfWeek() }

// GetMinimumDaysInFirstWeek is Info.getMinimumDaysInFirstWeek({locale}).
func (InfoType) GetMinimumDaysInFirstWeek(opts ...any) float64 {
	return infoLocale(opts).getMinDaysInFirstWeek()
}

// GetWeekendWeekdays is Info.getWeekendWeekdays({locale}).
func (InfoType) GetWeekendWeekdays(opts ...any) []float64 {
	return append([]float64{}, infoLocale(opts).getWeekendDays()...)
}

func infoMonths(args []any, format bool) []*string {
	length := argAt(args, 0)
	if isUndef(length) {
		length = "long"
	}
	o := optObj(args, 1)
	loc := createLocale(nullDefault(o.get("locale")), nullDefault(o.get("numberingSystem")),
		optDefault(o, "outputCalendar", "gregory"), nil, false)
	return stringList(loc.months(length, format))
}

// Months is Info.months(length = "long", {locale, numberingSystem, outputCalendar}).
func (InfoType) Months(args ...any) []*string { return infoMonths(args, false) }

// MonthsFormat is Info.monthsFormat(length = "long", {locale, numberingSystem, outputCalendar}).
func (InfoType) MonthsFormat(args ...any) []*string { return infoMonths(args, true) }

func infoWeekdays(args []any, format bool) []*string {
	length := argAt(args, 0)
	if isUndef(length) {
		length = "long"
	}
	o := optObj(args, 1)
	loc := createLocale(nullDefault(o.get("locale")), nullDefault(o.get("numberingSystem")), nil, nil, false)
	return stringList(loc.weekdays(length, format))
}

// Weekdays is Info.weekdays(length = "long", {locale, numberingSystem}).
func (InfoType) Weekdays(args ...any) []*string { return infoWeekdays(args, false) }

// WeekdaysFormat is Info.weekdaysFormat(length = "long", {locale, numberingSystem}).
func (InfoType) WeekdaysFormat(args ...any) []*string { return infoWeekdays(args, true) }

// Meridiems is Info.meridiems({locale}).
func (InfoType) Meridiems(opts ...any) []*string { return stringList(infoLocale(opts).meridiems()) }

// Eras is Info.eras(length = "short", {locale}).
func (InfoType) Eras(args ...any) []*string {
	length := argAt(args, 0)
	if isUndef(length) {
		length = "short"
	}
	loc := createLocale(nullDefault(optObj(args, 1).get("locale")), nil, "gregory", nil, false)
	return stringList(loc.eras(length))
}

// Features is Info.features().
func (InfoType) Features() Obj { return Obj{"relative", true, "localeWeek", true} }
