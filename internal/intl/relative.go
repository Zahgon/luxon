package intl

import (
	"math"
	"strings"
)

// RelativeTimeFormat mirrors Intl.RelativeTimeFormat for the recorded locales.
type RelativeTimeFormat struct {
	locale  string
	style   string
	numeric string
	nf      *NumberFormat
	plural  func(pluralOperands) string
}

var relUnits = map[string]string{
	"year": "years", "years": "years", "quarter": "quarters", "quarters": "quarters",
	"month": "months", "months": "months", "week": "weeks", "weeks": "weeks",
	"day": "days", "days": "days", "hour": "hours", "hours": "hours",
	"minute": "minutes", "minutes": "minutes", "second": "seconds", "seconds": "seconds",
}

// NewRelativeTimeFormat mirrors new Intl.RelativeTimeFormat(locale, {style, numeric}).
func NewRelativeTimeFormat(locale, style, numeric string) (*RelativeTimeFormat, error) {
	if style == "" {
		style = "long"
	}
	if numeric == "" {
		numeric = "always"
	}
	switch style {
	case "long", "short", "narrow":
	default:
		return nil, rangeError("Value " + style + " out of range for Intl.RelativeTimeFormat options property style")
	}
	switch numeric {
	case "always", "auto":
	default:
		return nil, rangeError("Value " + numeric + " out of range for Intl.RelativeTimeFormat options property numeric")
	}
	nf, err := NewNumberFormat(locale, NumberOptions{UseGrouping: true})
	if err != nil {
		return nil, err
	}
	if locale == "" {
		locale = SystemLocale()
	}
	if _, ok := locales().Rel[locale]; !ok {
		return nil, &Unsupported{What: "relative time patterns for " + locale}
	}
	pr, err := pluralRulesFor(locale)
	if err != nil {
		return nil, err
	}
	return &RelativeTimeFormat{locale: locale, style: style, numeric: numeric, nf: nf, plural: pr}, nil
}

// Format mirrors Intl.RelativeTimeFormat.prototype.format(value, unit).
func (r *RelativeTimeFormat) Format(v float64, unit string) (string, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "", rangeError("Value need to be finite number for Intl.RelativeTimeFormat.prototype.format()")
	}
	u, ok := relUnits[unit]
	if !ok {
		return "", rangeError("Invalid unit argument for Intl.RelativeTimeFormat.prototype.format() '" + unit + "'")
	}
	p := locales().Rel[r.locale][r.style+"|"+r.numeric+"|"+u]
	if p == nil {
		return "", &Unsupported{What: "relative time " + r.locale + " " + u}
	}
	negative := v < 0 || (v == 0 && math.Signbit(v))
	if r.numeric == "auto" && v == math.Trunc(v) && math.Abs(v) <= 2 {
		key := formatSmallInt(v)
		if phrase, ok := p.Auto[key]; ok {
			return phrase, nil
		}
	}
	text, op, _, _ := r.nf.magnitude(math.Abs(v))
	pats := p.Future
	if negative {
		pats = p.Past
	}
	pat, ok := pats[r.plural(op)]
	if !ok {
		pat = pats["other"]
	}
	return strings.Replace(pat, "{0}", text, 1), nil
}

func formatSmallInt(v float64) string {
	switch {
	case v == 0 && math.Signbit(v):
		return "-0"
	case v == 0:
		return "0"
	case v == 1:
		return "1"
	case v == 2:
		return "2"
	case v == -1:
		return "-1"
	}
	return "-2"
}
