package luxon

// A translator from JavaScript regular-expression source (non-unicode mode) to Go RE2 syntax,
// for the patterns Luxon builds: literals, escapes, classes, groups, alternation and greedy or
// lazy quantifiers. The "i" flag follows JavaScript's Canonicalize (toUpperCase, keeping a
// non-ASCII character that would uppercase to ASCII), not Go's Unicode case folding: under it
// "k" does not match the Kelvin sign and "s" does not match the long s.

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

type runeRange struct{ lo, hi rune }

var (
	jsRegexMu    sync.Mutex
	jsRegexCache = map[string]*regexp.Regexp{}
)

// compileJSRegex translates and compiles a JavaScript pattern; flags may contain "i".
func compileJSRegex(src string, flags string) *regexp.Regexp {
	key := flags + "/" + src
	jsRegexMu.Lock()
	re, ok := jsRegexCache[key]
	jsRegexMu.Unlock()
	if ok {
		return re
	}
	t := &jsTranslator{src: []rune(src), ignoreCase: strings.Contains(flags, "i")}
	goSrc, err := t.translate()
	if err == nil {
		re, err = regexp.Compile(goSrc)
	}
	if err != nil {
		throw("SyntaxError", "Invalid regular expression: /"+src+"/"+flags+": "+err.Error())
	}
	jsRegexMu.Lock()
	jsRegexCache[key] = re
	jsRegexMu.Unlock()
	return re
}

type jsTranslator struct {
	src        []rune
	pos        int
	ignoreCase bool
	out        strings.Builder
}

func (t *jsTranslator) peek() (rune, bool) {
	if t.pos < len(t.src) {
		return t.src[t.pos], true
	}
	return 0, false
}

func (t *jsTranslator) translate() (string, error) {
	for t.pos < len(t.src) {
		c := t.src[t.pos]
		switch c {
		case '^', '$', '|', ')':
			t.out.WriteRune(c)
			t.pos++
		case '(':
			t.pos++
			if t.pos+1 < len(t.src) && t.src[t.pos] == '?' {
				switch t.src[t.pos+1] {
				case ':':
					t.out.WriteString("(?:")
					t.pos += 2
				default:
					return "", fmt.Errorf("unsupported group (?%c", t.src[t.pos+1])
				}
			} else {
				t.out.WriteByte('(')
			}
		case '*', '+', '?':
			t.out.WriteRune(c)
			t.pos++
			t.lazy()
		case '{':
			if q, n := t.quantifier(); n > 0 {
				t.out.WriteString(q)
				t.pos += n
				t.lazy()
			} else {
				t.literal('{')
				t.pos++
			}
		case '}', ']':
			t.literal(c)
			t.pos++
		case '.':
			t.out.WriteString(`[^\n\r\x{2028}\x{2029}]`)
			t.pos++
		case '[':
			if err := t.class(); err != nil {
				return "", err
			}
		case '\\':
			if err := t.escape(); err != nil {
				return "", err
			}
		default:
			t.literal(c)
			t.pos++
		}
	}
	return t.out.String(), nil
}

func (t *jsTranslator) lazy() {
	if r, ok := t.peek(); ok && r == '?' {
		t.out.WriteByte('?')
		t.pos++
	}
}

// quantifier recognises {n}, {n,} or {n,m} at pos and returns it with its length in runes.
func (t *jsTranslator) quantifier() (string, int) {
	i := t.pos + 1
	start := i
	for i < len(t.src) && t.src[i] >= '0' && t.src[i] <= '9' {
		i++
	}
	if i == start {
		return "", 0
	}
	if i < len(t.src) && t.src[i] == '}' {
		return string(t.src[t.pos : i+1]), i + 1 - t.pos
	}
	if i >= len(t.src) || t.src[i] != ',' {
		return "", 0
	}
	i++
	for i < len(t.src) && t.src[i] >= '0' && t.src[i] <= '9' {
		i++
	}
	if i < len(t.src) && t.src[i] == '}' {
		return string(t.src[t.pos : i+1]), i + 1 - t.pos
	}
	return "", 0
}

// escape handles a backslash sequence outside a class.
func (t *jsTranslator) escape() error {
	t.pos++
	c, ok := t.peek()
	if !ok {
		return fmt.Errorf("\\ at end of pattern")
	}
	switch c {
	case 'd', 'D', 's', 'S', 'w', 'W':
		t.pos++
		set, neg := classEscape(c)
		t.writeClass(set, neg)
		return nil
	case 'b':
		t.pos++
		t.out.WriteString(`\b`)
		return nil
	case 'B':
		t.pos++
		t.out.WriteString(`\B`)
		return nil
	}
	r, err := t.characterEscape()
	if err != nil {
		return err
	}
	t.literal(r)
	return nil
}

// characterEscape reads the character an escape (after the backslash) stands for.
func (t *jsTranslator) characterEscape() (rune, error) {
	c := t.src[t.pos]
	t.pos++
	switch c {
	case 't':
		return '\t', nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 'v':
		return '\v', nil
	case 'f':
		return '\f', nil
	case '0':
		return 0, nil
	case 'u':
		if t.pos+4 <= len(t.src) {
			if v, err := strconv.ParseUint(string(t.src[t.pos:t.pos+4]), 16, 32); err == nil {
				t.pos += 4
				return rune(v), nil
			}
		}
		return 'u', nil
	case 'x':
		if t.pos+2 <= len(t.src) {
			if v, err := strconv.ParseUint(string(t.src[t.pos:t.pos+2]), 16, 32); err == nil {
				t.pos += 2
				return rune(v), nil
			}
		}
		return 'x', nil
	case 'c':
		if t.pos < len(t.src) {
			l := t.src[t.pos]
			if (l >= 'a' && l <= 'z') || (l >= 'A' && l <= 'Z') {
				t.pos++
				return l % 32, nil
			}
		}
		t.pos--
		return '\\', nil
	}
	if c >= '1' && c <= '9' {
		return 0, fmt.Errorf("backreferences are not supported")
	}
	return c, nil // identity escape
}

var (
	jsSpaceRanges = []runeRange{{9, 13}, {32, 32}, {0xA0, 0xA0}, {0x1680, 0x1680}, {0x2000, 0x200A},
		{0x2028, 0x2029}, {0x202F, 0x202F}, {0x205F, 0x205F}, {0x3000, 0x3000}, {0xFEFF, 0xFEFF}}
	jsDigitRanges = []runeRange{{'0', '9'}}
	jsWordRanges  = []runeRange{{'0', '9'}, {'A', 'Z'}, {'_', '_'}, {'a', 'z'}}
)

func classEscape(c rune) ([]runeRange, bool) {
	switch c {
	case 'd':
		return jsDigitRanges, false
	case 'D':
		return jsDigitRanges, true
	case 's':
		return jsSpaceRanges, false
	case 'S':
		return jsSpaceRanges, true
	case 'w':
		return jsWordRanges, false
	}
	return jsWordRanges, true
}

// class translates a [...] character class.
func (t *jsTranslator) class() error {
	t.pos++ // [
	negated := false
	if r, ok := t.peek(); ok && r == '^' {
		negated = true
		t.pos++
	}
	var set []runeRange
	for {
		c, ok := t.peek()
		if !ok {
			return fmt.Errorf("unterminated character class")
		}
		if c == ']' {
			t.pos++
			break
		}
		lo, loSet, err := t.classAtom()
		if err != nil {
			return err
		}
		if loSet != nil {
			set = append(set, loSet...)
			continue
		}
		// a range lo-hi?
		if t.pos+1 < len(t.src) && t.src[t.pos] == '-' && t.src[t.pos+1] != ']' {
			save := t.pos
			t.pos++
			hi, hiSet, err := t.classAtom()
			if err != nil {
				return err
			}
			if hiSet != nil {
				// [a-\d] is "a", "-" and the digits (Annex B)
				set = append(set, runeRange{lo, lo}, runeRange{'-', '-'})
				set = append(set, hiSet...)
				continue
			}
			if hi < lo {
				t.pos = save
				return fmt.Errorf("range out of order in character class")
			}
			set = append(set, runeRange{lo, hi})
			continue
		}
		set = append(set, runeRange{lo, lo})
	}
	t.writeClass(set, negated)
	return nil
}

// classAtom reads one class member: a character, or a class escape (returned as a set).
func (t *jsTranslator) classAtom() (rune, []runeRange, error) {
	c := t.src[t.pos]
	if c != '\\' {
		t.pos++
		return c, nil, nil
	}
	t.pos++
	if t.pos >= len(t.src) {
		return 0, nil, fmt.Errorf("\\ at end of pattern")
	}
	switch e := t.src[t.pos]; e {
	case 'd', 'D', 's', 'S', 'w', 'W':
		t.pos++
		set, neg := classEscape(e)
		if neg {
			set = negateRanges(set)
		}
		return 0, set, nil
	case 'b':
		t.pos++
		return '\b', nil, nil
	case '-':
		t.pos++
		return '-', nil, nil
	}
	r, err := t.characterEscape()
	return r, nil, err
}

// literal writes one character, expanded to its case-insensitive class under "i".
func (t *jsTranslator) literal(r rune) {
	if t.ignoreCase {
		if eq := caseClass(r); len(eq) > 1 {
			t.writeClass(eq, false)
			return
		}
	}
	t.out.WriteString(regexp.QuoteMeta(string(r)))
}

func (t *jsTranslator) writeClass(set []runeRange, negated bool) {
	if t.ignoreCase {
		set = expandCaseRanges(set)
	}
	set = normalizeRanges(set)
	t.out.WriteByte('[')
	if negated {
		t.out.WriteByte('^')
	}
	if len(set) == 0 {
		// An empty class matches nothing; its negation matches everything.
		if negated {
			t.out.WriteString(`\x{0}-\x{10FFFF}`)
			t.out.WriteByte(']')
			return
		}
		t.out.WriteString(`^\x{0}-\x{10FFFF}`)
		t.out.WriteByte(']')
		return
	}
	for _, r := range set {
		if r.lo == r.hi {
			fmt.Fprintf(&t.out, `\x{%X}`, r.lo)
		} else {
			fmt.Fprintf(&t.out, `\x{%X}-\x{%X}`, r.lo, r.hi)
		}
	}
	t.out.WriteByte(']')
}

func normalizeRanges(set []runeRange) []runeRange {
	if len(set) == 0 {
		return nil
	}
	s := append([]runeRange(nil), set...)
	sort.Slice(s, func(i, j int) bool { return s[i].lo < s[j].lo })
	out := []runeRange{s[0]}
	for _, r := range s[1:] {
		last := &out[len(out)-1]
		if r.lo <= last.hi+1 {
			if r.hi > last.hi {
				last.hi = r.hi
			}
			continue
		}
		out = append(out, r)
	}
	return out
}

func negateRanges(set []runeRange) []runeRange {
	set = normalizeRanges(set)
	var out []runeRange
	next := rune(0)
	for _, r := range set {
		if r.lo > next {
			out = append(out, runeRange{next, r.lo - 1})
		}
		next = r.hi + 1
	}
	if next <= utf8.MaxRune {
		out = append(out, runeRange{next, utf8.MaxRune})
	}
	return out
}

// ---------------------------------------------------------------------------------------------
// JavaScript case canonicalization (non-unicode mode)

var (
	canonOnce sync.Once
	canonOf   []rune          // Canonicalize(c) for every BMP code unit
	canonInv  map[rune][]rune // canonical value -> the BMP characters with that value
)

func initCanon() {
	canonOf = make([]rune, 0x10000)
	canonInv = make(map[rune][]rune)
	for c := rune(0); c < 0x10000; c++ {
		k := canonicalize(c)
		canonOf[c] = k
		if k != c || len(canonInv[k]) > 0 {
			canonInv[k] = append(canonInv[k], c)
		}
	}
	// make sure every canonical value lists itself
	for k, list := range canonInv {
		found := false
		for _, c := range list {
			if c == k {
				found = true
				break
			}
		}
		if !found && k < 0x10000 && canonOf[k] == k {
			canonInv[k] = append(canonInv[k], k)
		}
	}
}

func canonicalize(c rune) rune {
	if c >= 0xD800 && c <= 0xDFFF {
		return c
	}
	u := jsToUpperRune(c)
	if utf16Len(u) != 1 {
		return c
	}
	cu, _ := utf8.DecodeRuneInString(u)
	if c >= 128 && cu < 128 {
		return c
	}
	return cu
}

// caseClass is the set of characters that match r under the "i" flag.
func caseClass(r rune) []runeRange {
	if r >= 0x10000 {
		return []runeRange{{r, r}}
	}
	canonOnce.Do(initCanon)
	k := canonOf[r]
	var out []runeRange
	for _, c := range canonInv[k] {
		out = append(out, runeRange{c, c})
	}
	if len(out) == 0 {
		out = []runeRange{{r, r}}
	}
	return out
}

// expandCaseRanges adds every character whose canonical value equals that of a member.
func expandCaseRanges(set []runeRange) []runeRange {
	canonOnce.Do(initCanon)
	out := append([]runeRange(nil), set...)
	seen := map[rune]bool{}
	for _, r := range normalizeRanges(set) {
		lo, hi := r.lo, r.hi
		if lo >= 0x10000 {
			continue
		}
		if hi >= 0x10000 {
			hi = 0xFFFF
		}
		for c := lo; c <= hi; c++ {
			k := canonOf[c]
			if seen[k] {
				continue
			}
			seen[k] = true
			for _, d := range canonInv[k] {
				out = append(out, runeRange{d, d})
			}
		}
	}
	return out
}

// jsExec is RegExp.prototype.exec for a non-global pattern: the capture groups (nil for a
// group that did not participate), or nil when there is no match.
func jsExec(re *regexp.Regexp, s string) []*string { return submatches(re, s) }
