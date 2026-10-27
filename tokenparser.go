package luxon

// impl/tokenParser.js

import (
	"regexp"
	"strings"

	"github.com/moment/luxon-go/internal/intl"
)

const missingFTP = "missing Intl.DateTimeFormat.formatToParts support"

// tokenUnit is one parsing unit: the JavaScript source of its pattern and how to read its match.
type tokenUnit struct {
	regex         string
	deser         func(m []*string) any
	groups        int
	literal       bool
	token         fmtToken
	invalidReason string
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// matchValue is a capture group as JavaScript sees it: a string, or undefined.
func matchValue(s *string) any {
	if s == nil {
		return Undefined
	}
	return *s
}

func intUnit(regex string, post func(float64) float64) *tokenUnit {
	return &tokenUnit{regex: regex, deser: func(m []*string) any {
		v := parseDigits(derefStr(group(m, 0)))
		if post != nil {
			v = post(v)
		}
		return v
	}}
}

// fixListRegex makes dots optional and literal, and turns whitespace into \s.
func fixListRegex(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '.':
			b.WriteString(`\.?`)
		case isJSWhitespace(r):
			b.WriteString(`\s`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func stripInsensitivities(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '.':
		case isJSWhitespace(r):
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return jsToLower(b.String())
}

// regexSource is RegExp(src).source for the sources built here: the empty pattern reads (?:).
func regexSource(src string) string {
	if src == "" {
		return "(?:)"
	}
	return src
}

func listElement(v any) string {
	s, ok := v.(string)
	if !ok {
		throw("TypeError", "Cannot read properties of "+toJSString(v)+" (reading 'replace')")
	}
	return s
}

func oneOf(strs []any, startIndex float64) *tokenUnit {
	if strs == nil {
		return nil
	}
	srcs := make([]string, len(strs))
	for i, s := range strs {
		srcs[i] = fixListRegex(listElement(s))
	}
	return &tokenUnit{regex: regexSource(strings.Join(srcs, "|")), deser: func(m []*string) any {
		s := stripInsensitivities(derefStr(group(m, 0)))
		for i, x := range strs {
			if stripInsensitivities(x.(string)) == s {
				return float64(i) + startIndex
			}
		}
		return -1 + startIndex
	}}
}

func offsetUnit(regex string, groups int) *tokenUnit {
	return &tokenUnit{regex: regex, groups: groups, deser: func(m []*string) any {
		return signedOffset(group(m, 1), group(m, 2))
	}}
}

func simpleUnit(regex string) *tokenUnit {
	return &tokenUnit{regex: regex, deser: func(m []*string) any { return matchValue(group(m, 0)) }}
}

// escapeToken backslash-escapes - [ ] { } ( ) * + ? . , \ ^ $ | # and whitespace.
func escapeToken(value string) string {
	var b strings.Builder
	for _, r := range value {
		if strings.ContainsRune(`-[]{}()*+?.,\^$|#`, r) || isJSWhitespace(r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func unitForToken(token fmtToken, loc *locale) *tokenUnit {
	one := digitRegex(loc, "")
	two := digitRegex(loc, "{2}")
	three := digitRegex(loc, "{3}")
	four := digitRegex(loc, "{4}")
	six := digitRegex(loc, "{6}")
	oneOrTwo := digitRegex(loc, "{1,2}")
	oneToThree := digitRegex(loc, "{1,3}")
	oneToSix := digitRegex(loc, "{1,6}")
	oneToNine := digitRegex(loc, "{1,9}")
	twoToFour := digitRegex(loc, "{2,4}")
	fourToSix := digitRegex(loc, "{4,6}")
	literal := func(t fmtToken) *tokenUnit {
		return &tokenUnit{regex: regexSource(escapeToken(t.val)), literal: true, deser: func(m []*string) any {
			return matchValue(group(m, 0))
		}}
	}
	unitate := func(t fmtToken) *tokenUnit {
		if token.literal {
			return literal(t)
		}
		switch t.val {
		case "G":
			return oneOf(loc.eras("short"), 0)
		case "GG":
			return oneOf(loc.eras("long"), 0)
		case "y":
			return intUnit(oneToSix, nil)
		case "yy":
			return intUnit(twoToFour, untruncateYear)
		case "yyyy":
			return intUnit(four, nil)
		case "yyyyy":
			return intUnit(fourToSix, nil)
		case "yyyyyy":
			return intUnit(six, nil)
		case "M", "L", "d", "H", "h", "m", "q", "s", "W":
			return intUnit(oneOrTwo, nil)
		case "MM", "LL", "dd", "HH", "hh", "mm", "qq", "ss", "WW":
			return intUnit(two, nil)
		case "MMM":
			return oneOf(loc.months("short", true), 1)
		case "MMMM":
			return oneOf(loc.months("long", true), 1)
		case "LLL":
			return oneOf(loc.months("short", false), 1)
		case "LLLL":
			return oneOf(loc.months("long", false), 1)
		case "o", "S":
			return intUnit(oneToThree, nil)
		case "ooo", "SSS":
			return intUnit(three, nil)
		case "u":
			return simpleUnit(oneToNine)
		case "uu":
			return simpleUnit(oneOrTwo)
		case "uuu", "E", "c":
			return intUnit(one, nil)
		case "a":
			return oneOf(loc.meridiems(), 0)
		case "kkkk":
			return intUnit(four, nil)
		case "kk":
			return intUnit(twoToFour, untruncateYear)
		case "EEE":
			return oneOf(loc.weekdays("short", false), 1)
		case "EEEE":
			return oneOf(loc.weekdays("long", false), 1)
		case "ccc":
			return oneOf(loc.weekdays("short", true), 1)
		case "cccc":
			return oneOf(loc.weekdays("long", true), 1)
		case "Z", "ZZ":
			return offsetUnit("([+-]"+oneOrTwo+")(?::("+two+"))?", 2)
		case "ZZZ":
			return offsetUnit("([+-]"+oneOrTwo+")("+two+")?", 2)
		case "z":
			return simpleUnit(`[a-z_+-/]{1,256}?`)
		case " ":
			return simpleUnit(`[^\S\n\r]`)
		}
		return literal(t)
	}
	unit := unitate(token)
	if unit == nil {
		unit = &tokenUnit{invalidReason: missingFTP}
	}
	unit.token = token
	return unit
}

var partTypeStyleToTokenVal = map[string]any{
	"year":         map[string]string{"2-digit": "yy", "numeric": "yyyyy"},
	"month":        map[string]string{"numeric": "M", "2-digit": "MM", "short": "MMM", "long": "MMMM"},
	"day":          map[string]string{"numeric": "d", "2-digit": "dd"},
	"weekday":      map[string]string{"short": "EEE", "long": "EEEE"},
	"dayperiod":    "a",
	"dayPeriod":    "a",
	"hour12":       map[string]string{"numeric": "h", "2-digit": "hh"},
	"hour24":       map[string]string{"numeric": "H", "2-digit": "HH"},
	"minute":       map[string]string{"numeric": "m", "2-digit": "mm"},
	"second":       map[string]string{"numeric": "s", "2-digit": "ss"},
	"timeZoneName": map[string]string{"long": "ZZZZZ", "short": "ZZZ"},
}

// tokenForPart maps a formatToParts part to a format token; nil is undefined.
func tokenForPart(part intl.Part, formatOpts *jsObj, resolved intl.ResolvedDateTimeOptions) *fmtToken {
	if part.Type == "literal" {
		isSpace := isAllWhitespace(part.Value)
		val := part.Value
		if isSpace {
			val = " "
		}
		return &fmtToken{literal: !isSpace, val: val}
	}
	style := formatOpts.get(part.Type)
	actualType := part.Type
	if part.Type == "hour" {
		if h12 := formatOpts.get("hour12"); !isNullish(h12) {
			actualType = "hour24"
			if truthy(h12) {
				actualType = "hour12"
			}
		} else if hc := formatOpts.get("hourCycle"); !isNullish(hc) {
			actualType = "hour24"
			if hc == "h11" || hc == "h12" {
				actualType = "hour12"
			}
		} else {
			actualType = "hour24"
			if resolved.Hour12 != nil && *resolved.Hour12 {
				actualType = "hour12"
			}
		}
	}
	switch v := partTypeStyleToTokenVal[actualType].(type) {
	case string:
		return &fmtToken{val: v}
	case map[string]string:
		if t := v[toJSString(style)]; t != "" {
			return &fmtToken{val: t}
		}
	}
	return nil
}

var dummyDateTimeCache *DateTime

func getDummyDateTime() *DateTime {
	if dummyDateTimeCache == nil {
		dummyDateTimeCache = DateTimeFromMillis(1555555555555)
	}
	return dummyDateTimeCache
}

// formatOptsToTokens returns nil (null) without format options; an element is nil where the
// part has no token (undefined).
func formatOptsToTokens(formatOpts Obj, loc *locale) []*fmtToken {
	if formatOpts == nil {
		return nil
	}
	opts := objFrom(formatOpts)
	df := newFormatter(loc, opts).dtFormatter(getDummyDateTime(), nil)
	parts := df.formatToParts()
	resolved := df.resolvedOptions()
	tokens := make([]*fmtToken, len(parts))
	for i, p := range parts {
		tokens[i] = tokenForPart(p, opts, resolved)
	}
	return tokens
}

func maybeExpandMacroToken(token fmtToken, loc *locale) []fmtToken {
	if token.literal {
		return []fmtToken{token}
	}
	formatOpts, ok := macroTokenToFormatOpts[token.val]
	if !ok {
		return []fmtToken{token}
	}
	tokens := formatOptsToTokens(formatOpts, loc)
	out := make([]fmtToken, 0, len(tokens))
	for _, t := range tokens {
		if t == nil {
			return []fmtToken{token}
		}
		out = append(out, *t)
	}
	return out
}

func expandMacroTokens(tokens []fmtToken, loc *locale) []fmtToken {
	var out []fmtToken
	for _, t := range tokens {
		out = append(out, maybeExpandMacroToken(t, loc)...)
	}
	return out
}

// TokenParser is a format string compiled for parsing, as returned by DateTimeBuildFormatParser.
type TokenParser struct {
	locale            *locale
	format            string
	tokens            []fmtToken
	units             []*tokenUnit
	disqualifyingUnit *tokenUnit
	regexSrc          string
	regex             *regexp.Regexp
	handlers          []*tokenUnit
}

func newTokenParser(loc *locale, format string) *TokenParser {
	p := &TokenParser{locale: loc, format: format}
	p.tokens = expandMacroTokens(parseFormat(format), loc)
	for _, t := range p.tokens {
		p.units = append(p.units, unitForToken(t, loc))
	}
	for _, u := range p.units {
		if u.invalidReason != "" {
			p.disqualifyingUnit = u
			break
		}
	}
	if p.disqualifyingUnit == nil {
		var b strings.Builder
		b.WriteByte('^')
		for _, u := range p.units {
			b.WriteString("(" + u.regex + ")")
		}
		b.WriteByte('$')
		p.regexSrc = b.String()
		p.regex = compileJSRegex(p.regexSrc, "i")
		p.handlers = p.units
	}
	return p
}

// Format is the format string the parser was built from.
func (p *TokenParser) Format() string { return p.format }

// IsValid reports whether every token of the format can be parsed.
func (p *TokenParser) IsValid() bool { return p.disqualifyingUnit == nil }

// InvalidReason is nil for a valid parser.
func (p *TokenParser) InvalidReason() *string {
	if p.disqualifyingUnit == nil {
		return nil
	}
	return strp(p.disqualifyingUnit.invalidReason)
}

func (p *TokenParser) match(input string) ([]*string, *jsObj) {
	m := jsExec(p.regex, input)
	all := newObj()
	if m == nil {
		return nil, all
	}
	matchIndex := 1
	for _, h := range p.handlers {
		groups := 1
		if h.groups > 0 {
			groups = h.groups + 1
		}
		if !h.literal {
			end := matchIndex + groups
			if end > len(m) {
				end = len(m)
			}
			var slice []*string
			if matchIndex < end {
				slice = m[matchIndex:end]
			}
			all.set(h.token.val[:1], h.deser(slice))
		}
		matchIndex += groups
	}
	return m, all
}

func tokenField(token string) string {
	switch token {
	case "S":
		return "millisecond"
	case "s":
		return "second"
	case "m":
		return "minute"
	case "h", "H":
		return "hour"
	case "d":
		return "day"
	case "o":
		return "ordinal"
	case "L", "M":
		return "month"
	case "y":
		return "year"
	case "E", "c":
		return "weekday"
	case "W":
		return "weekNumber"
	case "k":
		return "weekYear"
	case "q":
		return "quarter"
	}
	return ""
}

func dateTimeFromMatches(matches *jsObj) (*jsObj, Zone, any, *string) {
	var zone Zone
	var specificOffset any = Undefined
	if z := matches.get("z"); !isUndef(z) {
		zone = IANAZoneCreate(toJSString(z))
	}
	if off := matches.get("Z"); !isUndef(off) {
		if zone == nil {
			zone = NewFixedOffsetZone(toNumber(off))
		}
		specificOffset = off
	}
	if q := matches.get("q"); !isUndef(q) {
		matches.set("M", float64((toNumber(q)-1)*3)+1)
	}
	var hourInvalidReason *string
	if hv := matches.get("h"); !isUndef(hv) {
		h := toNumber(hv)
		a := matches.get("a")
		af, aNum := a.(float64)
		switch {
		case !isUndef(a) && (h < 1 || h > 12):
			hourInvalidReason = strp(`the 12-hour value "` + numberToString(h) + `" is not in the [1, 12] range`)
		case h < 12 && aNum && af == 1:
			matches.set("h", h+12)
		case h == 12 && aNum && af == 0:
			matches.set("h", 0.0)
		}
	}
	if g, ok := matches.get("G").(float64); ok && g == 0 && truthy(matches.get("y")) {
		matches.set("y", -toNumber(matches.get("y")))
	}
	if u := matches.get("u"); !isUndef(u) {
		s := toJSString(u)
		if ms, ok := parseMillis(&s); ok {
			matches.set("S", ms)
		} else {
			matches.set("S", Undefined)
		}
	}
	vals := newObj()
	for _, k := range matches.keys {
		if f := tokenField(k); f != "" {
			vals.set(f, matches.vals[k])
		}
	}
	return vals, zone, specificOffset, hourInvalidReason
}

// explanation is the result of TokenParser.explainFromTokens.
type explanation struct {
	input          string
	valid          bool
	rawMatches     []*string
	matches        *jsObj
	result         *jsObj
	zone           Zone
	specificOffset any
	invalidReason  *string
}

func (p *TokenParser) explain(input string) *explanation {
	if !p.IsValid() {
		return &explanation{input: input, specificOffset: Undefined, invalidReason: p.InvalidReason()}
	}
	raw, matches := p.match(input)
	result, zone, specificOffset, reason := dateTimeFromMatches(matches)
	if matches.has("a") && matches.has("H") {
		throwConflictingSpecification("Can't include meridiem when specifying 24-hour format")
	}
	return &explanation{input: input, valid: true, rawMatches: raw, matches: matches, result: result,
		zone: zone, specificOffset: specificOffset, invalidReason: reason}
}

func tokensObj(tokens []fmtToken) []any {
	out := make([]any, len(tokens))
	for i, t := range tokens {
		out[i] = Obj{"literal", t.literal, "val", t.val}
	}
	return out
}

// regexString is RegExp.prototype.toString of the parser's pattern.
func (p *TokenParser) regexString() string {
	var b strings.Builder
	inClass, escaped := false, false
	for _, r := range p.regexSrc {
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '[':
			inClass = true
		case r == ']':
			inClass = false
		case r == '/' && !inClass:
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return "/" + b.String() + "/i"
}

func (p *TokenParser) explanationObj(e *explanation) Obj {
	if !e.valid {
		return Obj{"input", e.input, "tokens", tokensObj(p.tokens), "invalidReason", ptrAny(e.invalidReason)}
	}
	var raw any
	if e.rawMatches != nil {
		r := make([]any, len(e.rawMatches))
		for i, s := range e.rawMatches {
			r[i] = matchValue(s)
		}
		raw = r
	}
	var zone any
	if e.zone != nil {
		zone = e.zone
	}
	reason := any(Undefined)
	if e.invalidReason != nil {
		reason = *e.invalidReason
	}
	return Obj{"input", e.input, "tokens", tokensObj(p.tokens), "regex", p.regexString(), "rawMatches", raw,
		"matches", e.matches.toObj(), "result", e.result.toObj(), "zone", zone,
		"specificOffset", e.specificOffset, "invalidReason", reason}
}

// ExplainFromTokens is TokenParser.explainFromTokens(input).
func (p *TokenParser) ExplainFromTokens(input string) Obj { return p.explanationObj(p.explain(input)) }
