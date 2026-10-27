package luxon

// The Luxon error classes. Each is raised as a panic carrying *Error with the class name.

func throwInvalidDateTime(reason *invalid) {
	throw("InvalidDateTimeError", "Invalid DateTime: "+reason.toMessage())
}

func throwInvalidInterval(reason *invalid) {
	throw("InvalidIntervalError", "Invalid Interval: "+reason.toMessage())
}

func throwInvalidDuration(reason *invalid) {
	throw("InvalidDurationError", "Invalid Duration: "+reason.toMessage())
}

func throwConflictingSpecification(msg string) { throw("ConflictingSpecificationError", msg) }

func throwInvalidUnit(unit any) { throw("InvalidUnitError", "Invalid unit "+toJSString(unit)) }

func throwInvalidArgument(msg string) { throw("InvalidArgumentError", msg) }

func throwZoneIsAbstract() { throw("ZoneIsAbstractError", "Zone is an abstract class") }

func throwRange(msg string) { throw("RangeError", msg) }

// invalid is impl/invalid.js: why a DateTime, Duration or Interval is invalid.
type invalid struct {
	reason      string
	explanation *string
}

func newInvalid(reason string, explanation *string) *invalid {
	return &invalid{reason: reason, explanation: explanation}
}

func (i *invalid) toMessage() string {
	if i.explanation != nil && *i.explanation != "" {
		return i.reason + ": " + *i.explanation
	}
	return i.reason
}

func strp(s string) *string { return &s }
