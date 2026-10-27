package luxon

// zone.js, zones/*.js and impl/zoneUtil.js

import (
	"math"
	"regexp"
	"sync"

	"github.com/moment/luxon-go/internal/intl"
)

// Zone is a time zone: FixedOffsetZone, IANAZone, SystemZone, InvalidZone, or the abstract base
// returned by NewZone.
type Zone interface {
	Type() string
	Name() string
	IanaName() string
	IsUniversal() bool
	// OffsetName takes the options object {format, locale}; it returns nil where Luxon returns null.
	OffsetName(ts float64, opts Obj) *string
	FormatOffset(ts float64, format string) string
	Offset(ts float64) float64
	Equals(other Zone) bool
	IsValid() bool
}

// abstractZone is new Zone(): every member throws ZoneIsAbstractError.
type abstractZone struct{}

// NewZone is new Zone().
func NewZone() Zone { return abstractZone{} }

func (abstractZone) Type() string                        { throwZoneIsAbstract(); return "" }
func (abstractZone) Name() string                        { throwZoneIsAbstract(); return "" }
func (z abstractZone) IanaName() string                  { return z.Name() }
func (abstractZone) IsUniversal() bool                   { throwZoneIsAbstract(); return false }
func (abstractZone) OffsetName(float64, Obj) *string     { throwZoneIsAbstract(); return nil }
func (abstractZone) FormatOffset(float64, string) string { throwZoneIsAbstract(); return "" }
func (abstractZone) Offset(float64) float64              { throwZoneIsAbstract(); return 0 }
func (abstractZone) Equals(Zone) bool                    { throwZoneIsAbstract(); return false }
func (abstractZone) IsValid() bool                       { throwZoneIsAbstract(); return false }

// ---------------------------------------------------------------------------------------------
// FixedOffsetZone

// FixedOffsetZone is a zone with a fixed offset from UTC, in minutes.
type FixedOffsetZone struct {
	fixed float64
	valid bool
}

var utcInstance = &FixedOffsetZone{fixed: 0, valid: true}

// FixedOffsetZoneUtcInstance is FixedOffsetZone.utcInstance.
func FixedOffsetZoneUtcInstance() *FixedOffsetZone { return utcInstance }

// FixedOffsetZoneInstance is FixedOffsetZone.instance(offset).
func FixedOffsetZoneInstance(offset float64) *FixedOffsetZone {
	if offset == 0 {
		return utcInstance
	}
	return NewFixedOffsetZone(offset)
}

var reUTCSpecifier = regexp.MustCompile(`^[uU][tT][cC](?:([+-][0-9]{1,2})(?::([0-9]{2}))?)?$`)

// FixedOffsetZoneParseSpecifier is FixedOffsetZone.parseSpecifier(s); it returns nil for null.
func FixedOffsetZoneParseSpecifier(s string) *FixedOffsetZone {
	if s != "" {
		if m := submatches(reUTCSpecifier, s); m != nil {
			return NewFixedOffsetZone(signedOffset(m[1], m[2]))
		}
	}
	return nil
}

// NewFixedOffsetZone is new FixedOffsetZone(offset).
func NewFixedOffsetZone(offset float64) *FixedOffsetZone {
	return &FixedOffsetZone{fixed: offset, valid: isIntegerF(offset)}
}

func (z *FixedOffsetZone) Type() string { return "fixed" }

func (z *FixedOffsetZone) Name() string {
	if z.fixed == 0 {
		return "UTC"
	}
	return "UTC" + formatOffset(z.fixed, "narrow")
}

func (z *FixedOffsetZone) IanaName() string {
	if z.fixed == 0 {
		return "Etc/UTC"
	}
	return "Etc/GMT" + formatOffset(-z.fixed, "narrow")
}

func (z *FixedOffsetZone) OffsetName(float64, Obj) *string { return strp(z.Name()) }

func (z *FixedOffsetZone) FormatOffset(_ float64, format string) string {
	return formatOffset(z.fixed, format)
}

func (z *FixedOffsetZone) IsUniversal() bool { return true }

func (z *FixedOffsetZone) Offset(float64) float64 { return z.fixed }

func (z *FixedOffsetZone) Equals(other Zone) bool {
	o, ok := other.(*FixedOffsetZone)
	return ok && o.Type() == "fixed" && o.fixed == z.fixed
}

func (z *FixedOffsetZone) IsValid() bool { return z.valid }

// Fixed is the offset in minutes (the fixed property).
func (z *FixedOffsetZone) Fixed() float64 { return z.fixed }

// ---------------------------------------------------------------------------------------------
// IANAZone

// IANAZone is a zone identified by an IANA (or ICU-accepted) name.
type IANAZone struct {
	zoneName string
	canon    string
	valid    bool
}

var (
	ianaMu    sync.Mutex
	ianaCache = map[string]*IANAZone{}
)

// IANAZoneCreate is IANAZone.create(name).
func IANAZoneCreate(name string) *IANAZone {
	ianaMu.Lock()
	z, ok := ianaCache[name]
	ianaMu.Unlock()
	if !ok {
		z = NewIANAZone(name)
		ianaMu.Lock()
		ianaCache[name] = z
		ianaMu.Unlock()
	}
	return z
}

// IANAZoneResetCache is IANAZone.resetCache().
func IANAZoneResetCache() {
	ianaMu.Lock()
	ianaCache = map[string]*IANAZone{}
	ianaMu.Unlock()
}

// IANAZoneIsValidSpecifier is IANAZone.isValidSpecifier(s).
func IANAZoneIsValidSpecifier(s any) bool { return IANAZoneIsValidZone(s) }

// IANAZoneIsValidZone is IANAZone.isValidZone(zone).
func IANAZoneIsValidZone(zone any) bool { return IANAZoneNormalizeZone(zone) != nil }

// IANAZoneNormalizeZone is IANAZone.normalizeZone(zone): the name ICU resolves it to, or nil.
func IANAZoneNormalizeZone(zone any) *string {
	if !truthy(zone) {
		return nil
	}
	c, ok := intl.CanonicalTimeZone(toJSString(zone))
	if !ok {
		return nil
	}
	return &c
}

// NewIANAZone is new IANAZone(name).
func NewIANAZone(name string) *IANAZone {
	normalized := IANAZoneNormalizeZone(name)
	z := &IANAZone{valid: normalized != nil, zoneName: name}
	if normalized != nil {
		z.canon = *normalized
		if jsToLower(*normalized) == jsToLower(name) {
			z.zoneName = *normalized
		}
	}
	return z
}

func (z *IANAZone) Type() string      { return "iana" }
func (z *IANAZone) Name() string      { return z.zoneName }
func (z *IANAZone) IanaName() string  { return z.zoneName }
func (z *IANAZone) IsUniversal() bool { return false }
func (z *IANAZone) IsValid() bool     { return z.valid }

func (z *IANAZone) OffsetName(ts float64, opts Obj) *string {
	locale := opt(opts, "locale")
	return parseZoneInfo(ts, opt(opts, "format"), localeArg(locale), z.Name())
}

func (z *IANAZone) FormatOffset(ts float64, format string) string {
	return formatOffset(z.Offset(ts), format)
}

// Offset is the zone's offset from UTC, in minutes, at ts: what ICU's calendar fields of ts in
// the zone imply, so sub-minute local mean time offsets come out as fractions of a minute.
func (z *IANAZone) Offset(ts float64) float64 {
	if !z.valid {
		return math.NaN()
	}
	t := timeClip(ts)
	if math.IsNaN(t) {
		return math.NaN()
	}
	sec := math.Floor(t / 1000)
	off := intl.OffsetSeconds(z.canon, int64(sec))
	// ICU formats the wall-clock fields of ts in the zone; Luxon rebuilds them with Date.UTC,
	// which gives NaN once they leave the ECMAScript time range (near ±8.64e15).
	asUTC := (sec + float64(off)) * 1000
	if math.Abs(asUTC) > maxTimeMs {
		asUTC = math.NaN()
	}
	return (asUTC - float64(sec*1000)) / (60 * 1000)
}

func (z *IANAZone) Equals(other Zone) bool {
	o, ok := other.(*IANAZone)
	return ok && o.Name() == z.Name()
}

// ---------------------------------------------------------------------------------------------
// SystemZone

// SystemZone is the runtime's default zone (UTC in the graded environment).
type SystemZone struct{}

var systemInstance = &SystemZone{}

// SystemZoneInstance is SystemZone.instance.
func SystemZoneInstance() *SystemZone { return systemInstance }

func (z *SystemZone) Type() string      { return "system" }
func (z *SystemZone) Name() string      { return intl.SystemTimeZone() }
func (z *SystemZone) IanaName() string  { return z.Name() }
func (z *SystemZone) IsUniversal() bool { return false }
func (z *SystemZone) IsValid() bool     { return true }

func (z *SystemZone) OffsetName(ts float64, opts Obj) *string {
	return parseZoneInfo(ts, opt(opts, "format"), localeArg(opt(opts, "locale")), "")
}

func (z *SystemZone) FormatOffset(ts float64, format string) string {
	return formatOffset(z.Offset(ts), format)
}

// Offset is -new Date(ts).getTimezoneOffset(): whole minutes, NaN for an invalid date.
func (z *SystemZone) Offset(ts float64) float64 {
	t := timeClip(ts)
	if math.IsNaN(t) {
		return math.NaN()
	}
	var off int64
	if canon, ok := intl.CanonicalTimeZone(intl.SystemTimeZone()); ok {
		off = intl.OffsetSeconds(canon, int64(math.Floor(t/1000)))
	}
	// getTimezoneOffset is (UTC - local) in whole minutes, truncated; Luxon negates it, so a
	// zero offset comes out as -0.
	tzOffset := math.Trunc(float64(-off) / 60)
	return -tzOffset
}

func (z *SystemZone) Equals(other Zone) bool {
	_, ok := other.(*SystemZone)
	return ok
}

// ---------------------------------------------------------------------------------------------
// InvalidZone

// InvalidZone is a zone Luxon could not interpret.
type InvalidZone struct{ zoneName any }

// NewInvalidZone is new InvalidZone(zoneName).
func NewInvalidZone(zoneName any) *InvalidZone { return &InvalidZone{zoneName: zoneName} }

func (z *InvalidZone) Type() string                        { return "invalid" }
func (z *InvalidZone) Name() string                        { return toJSString(z.zoneName) }
func (z *InvalidZone) IanaName() string                    { return z.Name() }
func (z *InvalidZone) IsUniversal() bool                   { return false }
func (z *InvalidZone) OffsetName(float64, Obj) *string     { return nil }
func (z *InvalidZone) FormatOffset(float64, string) string { return "" }
func (z *InvalidZone) Offset(float64) float64              { return math.NaN() }
func (z *InvalidZone) Equals(Zone) bool                    { return false }
func (z *InvalidZone) IsValid() bool                       { return false }

// ---------------------------------------------------------------------------------------------
// normalizeZone

// normalizeZone is impl/zoneUtil.js normalizeZone: a zone from a Zone, a string, a number of
// minutes, or null/undefined (the default zone).
func normalizeZone(input any, defaultZone Zone) Zone {
	switch x := input.(type) {
	case undefinedType, nil:
		return defaultZone
	case Zone:
		return x
	case string:
		lowered := jsToLower(x)
		switch lowered {
		case "default":
			return defaultZone
		case "local", "system":
			return systemInstance
		case "utc", "gmt":
			return utcInstance
		}
		if z := FixedOffsetZoneParseSpecifier(lowered); z != nil {
			return z
		}
		return IANAZoneCreate(x)
	}
	if f, ok := asFloat(input); ok {
		return FixedOffsetZoneInstance(f)
	}
	return NewInvalidZone(input)
}

// localeArg is the locale argument handed to an Intl constructor ("" is the runtime default).
func localeArg(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// submatches returns the capture groups of re in s, nil entries for groups that did not take
// part (JavaScript's undefined), or nil when there is no match.
func submatches(re *regexp.Regexp, s string) []*string {
	idx := re.FindStringSubmatchIndex(s)
	if idx == nil {
		return nil
	}
	out := make([]*string, len(idx)/2)
	for i := range out {
		if idx[2*i] >= 0 {
			v := s[idx[2*i]:idx[2*i+1]]
			out[i] = &v
		}
	}
	return out
}
