package intl

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ListFormat mirrors Intl.ListFormat for the recorded locales.
type ListFormat struct {
	two, start, middle, end string
	contextual              func(next string) bool
	twoAlt, endAlt          string
}

// NewListFormat mirrors new Intl.ListFormat(locale, {type, style}).
func NewListFormat(locale, typ, style string) (*ListFormat, error) {
	if typ == "" {
		typ = "conjunction"
	}
	if style == "" {
		style = "long"
	}
	switch typ {
	case "conjunction", "disjunction", "unit":
	default:
		return nil, rangeError("Value " + typ + " out of range for Intl.ListFormat options property type")
	}
	switch style {
	case "long", "short", "narrow":
	default:
		return nil, rangeError("Value " + style + " out of range for Intl.ListFormat options property style")
	}
	r, err := ResolveNumberFormat(locale)
	if err != nil {
		return nil, err
	}
	if locale == "" {
		locale = SystemLocale()
	}
	p, ok := locales().Lists[locale][typ+"|"+style]
	if !ok {
		return nil, &Unsupported{What: "list patterns for " + locale}
	}
	l := &ListFormat{two: p["two"], start: p["start"], middle: p["middle"], end: p["end"]}
	lang, _, _ := strings.Cut(r.Locale, "-")
	switch lang {
	case "es":
		switch {
		case l.two == "{0} y {1}" || l.end == "{0} y {1}":
			l.contextual, l.twoAlt, l.endAlt = changesToE, swapIf(l.two, "{0} y {1}", "{0} e {1}"), swapIf(l.end, "{0} y {1}", "{0} e {1}")
		case l.two == "{0} o {1}" || l.end == "{0} o {1}":
			l.contextual, l.twoAlt, l.endAlt = changesToU, swapIf(l.two, "{0} o {1}", "{0} u {1}"), swapIf(l.end, "{0} o {1}", "{0} u {1}")
		}
	case "he", "iw":
		const vav, vavDash = "{0} \u05D5{1}", "{0} \u05D5-{1}"
		if l.two == vavDash || l.end == vavDash {
			l.twoAlt, l.endAlt = l.two, l.end
			l.two, l.end = swapIf(l.two, vavDash, vav), swapIf(l.end, vavDash, vav)
			l.contextual = notHebrew
		}
	}
	return l, nil
}

func swapIf(p, from, to string) string {
	if p == from {
		return to
	}
	return p
}

func changesToE(s string) bool {
	if s == "" {
		return false
	}
	if (s[0] == 'h' || s[0] == 'H') && len(s) > 1 && (s[1] == 'i' || s[1] == 'I') &&
		(len(s) == 2 || !(s[2] == 'a' || s[2] == 'A' || s[2] == 'e' || s[2] == 'E')) {
		return true
	}
	return s[0] == 'i' || s[0] == 'I'
}

func changesToU(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == 'o' || s[0] == 'O' || s[0] == '8' {
		return true
	}
	if (s[0] == 'h' || s[0] == 'H') && len(s) > 1 && (s[1] == 'o' || s[1] == 'O') {
		return true
	}
	return len(s) >= 2 && s[0] == '1' && s[1] == '1' && (len(s) == 2 || s[2] == ' ')
}

func notHebrew(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return !unicode.Is(unicode.Hebrew, r)
}

func apply(p, a, b string) string {
	return strings.Replace(strings.Replace(p, "{0}", a, 1), "{1}", b, 1)
}

// Format mirrors Intl.ListFormat.prototype.format.
func (l *ListFormat) Format(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		p := l.two
		if l.contextual != nil && l.contextual(items[1]) {
			p = l.twoAlt
		}
		return apply(p, items[0], items[1])
	}
	out := apply(l.start, items[0], items[1])
	for _, it := range items[2 : len(items)-1] {
		out = apply(l.middle, out, it)
	}
	last := items[len(items)-1]
	p := l.end
	if l.contextual != nil && l.contextual(last) {
		p = l.endAlt
	}
	return apply(p, out, last)
}
