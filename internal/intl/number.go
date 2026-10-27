package intl

import (
	"math"
	"strconv"
	"strings"
)

// NumberOptions are the Intl.NumberFormat options Luxon passes.
type NumberOptions struct {
	UseGrouping          bool
	MinimumIntegerDigits int
	SignDisplay          string // auto, always, never, exceptZero, negative
	Unit                 string // style "unit" when set
	UnitDisplay          string // long, short, narrow
}

// NumberFormat formats numbers the way ICU does for the recorded locales.
type NumberFormat struct {
	locale string
	sym    *numSymbols
	opts   NumberOptions
	units  map[string]string
	plural func(op pluralOperands) string
}

var numberUnits = map[string]bool{"year": true, "month": true, "week": true, "day": true, "hour": true, "minute": true, "second": true, "millisecond": true}

// NewNumberFormat mirrors new Intl.NumberFormat(locale, options).
func NewNumberFormat(locale string, opts NumberOptions) (*NumberFormat, error) {
	sym, err := numberSymbols(locale)
	if err != nil {
		return nil, err
	}
	if locale == "" {
		locale = SystemLocale()
	}
	if opts.SignDisplay == "" {
		opts.SignDisplay = "auto"
	}
	if opts.MinimumIntegerDigits == 0 {
		opts.MinimumIntegerDigits = 1
	}
	if opts.MinimumIntegerDigits < 1 || opts.MinimumIntegerDigits > 21 {
		return nil, rangeError("minimumIntegerDigits value is out of range.")
	}
	f := &NumberFormat{locale: locale, sym: sym, opts: opts}
	if opts.Unit != "" {
		if !numberUnits[opts.Unit] {
			return nil, rangeError("Invalid unit argument for Intl.NumberFormat() '" + opts.Unit + "'")
		}
		if opts.UnitDisplay == "" {
			opts.UnitDisplay = "short"
			f.opts.UnitDisplay = "short"
		}
		u, ok := locales().Units[locale][opts.UnitDisplay+"|"+opts.Unit]
		if !ok {
			return nil, &Unsupported{What: "unit patterns for " + locale}
		}
		f.units = u
		pr, err := pluralRulesFor(locale)
		if err != nil {
			return nil, err
		}
		f.plural = pr
	}
	return f, nil
}

// decimalDigits is a non-negative decimal: digits with the decimal point after intLen digits.
type decimalDigits struct {
	digits string // no leading zeros except "0"
	intLen int
}

// roundDecimal converts |v| to its shortest round-trip decimal and rounds half away from zero
// to at most maxFrac fraction digits, as ICU does with its default precision.
func roundDecimal(v float64, maxFrac int) (intPart, fracPart string) {
	s := strconv.FormatFloat(math.Abs(v), 'e', -1, 64)
	mant, expStr, _ := strings.Cut(s, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := strings.Replace(mant, ".", "", 1)
	pointPos := exp + 1 // digits before the decimal point
	if pointPos <= 0 {
		digits = strings.Repeat("0", -pointPos+1) + digits
		pointPos = 1
	}
	if len(digits) < pointPos {
		digits += strings.Repeat("0", pointPos-len(digits))
	}
	keep := pointPos + maxFrac
	if len(digits) > keep {
		roundUp := digits[keep] >= '5'
		digits = digits[:keep]
		if roundUp {
			b := []byte(digits)
			i := len(b) - 1
			for ; i >= 0; i-- {
				if b[i] == '9' {
					b[i] = '0'
					continue
				}
				b[i]++
				break
			}
			digits = string(b)
			if i < 0 {
				digits = "1" + digits
				pointPos++
			}
		}
	}
	intPart = strings.TrimLeft(digits[:pointPos], "0")
	if intPart == "" {
		intPart = "0"
	}
	fracPart = strings.TrimRight(digits[pointPos:], "0")
	return
}

func (f *NumberFormat) localize(s string) string {
	if f.sym.NS == "latn" {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteString(f.sym.Digits[r-'0'])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (f *NumberFormat) group(intPart string) string {
	if !f.opts.UseGrouping || len(intPart) <= 3 {
		return intPart
	}
	if len(intPart) < 3+f.sym.MinGrouping {
		return intPart
	}
	head, tail := intPart[:len(intPart)-3], intPart[len(intPart)-3:]
	var groups []string
	for len(head) > f.sym.Secondary {
		groups = append([]string{head[len(head)-f.sym.Secondary:]}, groups...)
		head = head[:len(head)-f.sym.Secondary]
	}
	groups = append([]string{head}, groups...)
	return strings.Join(append(groups, tail), "\x00")
}

// formatParts returns the localized magnitude and whether the value is negative (or -0) and
// whether it rounds to zero.
func (f *NumberFormat) magnitude(v float64) (text string, op pluralOperands, negative, zero bool) {
	negative = v < 0 || (v == 0 && math.Signbit(v))
	if math.IsInf(v, 0) {
		return f.sym.Inf, pluralOperands{n: math.Inf(1)}, negative, false
	}
	ip, fp := roundDecimal(v, 3)
	digits := ip
	if pad := f.opts.MinimumIntegerDigits - len(ip); pad > 0 {
		digits = strings.Repeat("0", pad) + ip
	}
	g := f.group(digits)
	out := f.localize(g)
	out = strings.ReplaceAll(out, "\x00", f.sym.Group)
	if fp != "" {
		out += f.sym.Decimal + f.localize(fp)
	}
	zero = ip == "0" && fp == ""
	return out, operandsOf(ip, fp), negative, zero
}

func (f *NumberFormat) sign(negative, zero bool) string {
	switch f.opts.SignDisplay {
	case "never":
		return ""
	case "always":
		if negative {
			return f.sym.Minus
		}
		return f.sym.Plus
	case "exceptZero":
		if zero {
			return ""
		}
		if negative {
			return f.sym.Minus
		}
		return f.sym.Plus
	case "negative":
		if negative && !zero {
			return f.sym.Minus
		}
		return ""
	}
	if negative {
		return f.sym.Minus
	}
	return ""
}

// Format mirrors Intl.NumberFormat.prototype.format for a Number argument.
func (f *NumberFormat) Format(v float64) string {
	if math.IsNaN(v) {
		return f.applyUnit(f.sign(false, true)+f.sym.NaN, pluralOperands{nan: true})
	}
	text, op, negative, zero := f.magnitude(v)
	return f.applyUnit(f.sign(negative, zero)+text, op)
}

func (f *NumberFormat) applyUnit(num string, op pluralOperands) string {
	if f.units == nil {
		return num
	}
	cat := f.plural(op)
	pat, ok := f.units[cat]
	if !ok {
		pat = f.units["other"]
	}
	return strings.Replace(pat, "{0}", num, 1)
}
