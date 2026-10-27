package intl

import (
	"math"
	"strconv"
	"strings"
)

// pluralOperands are the CLDR operands of a formatted decimal: n absolute value, i integer
// digits (only the last 15 when larger), v visible fraction digits, f the fraction digits as an
// integer.
type pluralOperands struct {
	n   float64
	i   int64
	v   int
	f   int64
	nan bool
}

func operandsOf(intPart, fracPart string) pluralOperands {
	tail := intPart
	if len(tail) > 15 {
		tail = tail[len(tail)-15:]
	}
	i, _ := strconv.ParseInt(tail, 10, 64)
	var f int64
	if fracPart != "" {
		f, _ = strconv.ParseInt(fracPart, 10, 64)
	}
	n, _ := strconv.ParseFloat(intPart+"."+fracPart+"0", 64)
	return pluralOperands{n: n, i: i, v: len(fracPart), f: f}
}

func (o pluralOperands) frac() float64 { return float64(o.f) / math.Pow10(o.v) }

// nmod is n % m with the fraction kept, as CLDR defines it.
func (o pluralOperands) nmod(m int64) float64 { return float64(o.i%m) + o.frac() }

func in(x float64, lo, hi int64) bool {
	return x == math.Trunc(x) && x >= float64(lo) && x <= float64(hi)
}

func inI(x, lo, hi int64) bool { return x >= lo && x <= hi }

func millionMany(o pluralOperands) bool { return o.i != 0 && o.i%1000000 == 0 && o.v == 0 }

var pluralRuleSets = map[string]func(o pluralOperands) string{
	"am": oneIZeroOrN1, "bn": oneIZeroOrN1, "fa": oneIZeroOrN1, "hi": oneIZeroOrN1,
	"ar": func(o pluralOperands) string {
		switch {
		case o.n == 0:
			return "zero"
		case o.n == 1:
			return "one"
		case o.n == 2:
			return "two"
		case in(o.nmod(100), 3, 10):
			return "few"
		case in(o.nmod(100), 11, 99):
			return "many"
		}
		return "other"
	},
	"ca": oneI1V0Many, "it": oneI1V0Many,
	"cs": func(o pluralOperands) string {
		switch {
		case o.i == 1 && o.v == 0:
			return "one"
		case inI(o.i, 2, 4) && o.v == 0:
			return "few"
		case o.v != 0:
			return "many"
		}
		return "other"
	},
	"cy": func(o pluralOperands) string {
		switch o.n {
		case 0:
			return "zero"
		case 1:
			return "one"
		case 2:
			return "two"
		case 3:
			return "few"
		case 6:
			return "many"
		}
		return "other"
	},
	"de": oneI1V0, "en": oneI1V0, "fi": oneI1V0, "nl": oneI1V0, "sv": oneI1V0,
	"en-US": oneI1V0, "en-u-va-posix": oneI1V0,
	"el": oneN1, "hu": oneN1, "mr": oneN1, "no": oneN1, "tr": oneN1,
	"es": func(o pluralOperands) string {
		switch {
		case o.n == 1:
			return "one"
		case millionMany(o):
			return "many"
		}
		return "other"
	},
	"fr": func(o pluralOperands) string {
		switch {
		case o.i == 0 || o.i == 1:
			return "one"
		case millionMany(o):
			return "many"
		}
		return "other"
	},
	"pt": func(o pluralOperands) string {
		switch {
		case inI(o.i, 0, 1):
			return "one"
		case millionMany(o):
			return "many"
		}
		return "other"
	},
	"he": func(o pluralOperands) string {
		switch {
		case o.i == 1 && o.v == 0 || o.i == 0 && o.v != 0:
			return "one"
		case o.i == 2 && o.v == 0:
			return "two"
		}
		return "other"
	},
	"id": otherOnly, "ja": otherOnly, "ko": otherOnly, "my": otherOnly, "th": otherOnly, "vi": otherOnly, "zh": otherOnly,
	"lt": func(o pluralOperands) string {
		n10, n100 := o.nmod(10), o.nmod(100)
		switch {
		case n10 == 1 && !in(n100, 11, 19):
			return "one"
		case in(n10, 2, 9) && !in(n100, 11, 19):
			return "few"
		case o.f != 0:
			return "many"
		}
		return "other"
	},
	"pl": func(o pluralOperands) string {
		i10, i100 := o.i%10, o.i%100
		switch {
		case o.i == 1 && o.v == 0:
			return "one"
		case o.v == 0 && inI(i10, 2, 4) && !inI(i100, 12, 14):
			return "few"
		case o.v == 0 && o.i != 1 && inI(i10, 0, 1) || o.v == 0 && inI(i10, 5, 9) || o.v == 0 && inI(i100, 12, 14):
			return "many"
		}
		return "other"
	},
	"ru": eastSlavic, "uk": eastSlavic,
	"sr": func(o pluralOperands) string {
		i10, i100, f10, f100 := o.i%10, o.i%100, o.f%10, o.f%100
		switch {
		case o.v == 0 && i10 == 1 && i100 != 11 || f10 == 1 && f100 != 11:
			return "one"
		case o.v == 0 && inI(i10, 2, 4) && !inI(i100, 12, 14) || inI(f10, 2, 4) && !inI(f100, 12, 14):
			return "few"
		}
		return "other"
	},
}

func otherOnly(pluralOperands) string { return "other" }

func oneI1V0(o pluralOperands) string {
	if o.i == 1 && o.v == 0 {
		return "one"
	}
	return "other"
}

func oneN1(o pluralOperands) string {
	if o.n == 1 {
		return "one"
	}
	return "other"
}

func oneIZeroOrN1(o pluralOperands) string {
	if o.i == 0 || o.n == 1 {
		return "one"
	}
	return "other"
}

func oneI1V0Many(o pluralOperands) string {
	switch {
	case o.i == 1 && o.v == 0:
		return "one"
	case millionMany(o):
		return "many"
	}
	return "other"
}

func eastSlavic(o pluralOperands) string {
	i10, i100 := o.i%10, o.i%100
	switch {
	case o.v == 0 && i10 == 1 && i100 != 11:
		return "one"
	case o.v == 0 && inI(i10, 2, 4) && !inI(i100, 12, 14):
		return "few"
	case o.v == 0 && i10 == 0 || o.v == 0 && inI(i10, 5, 9) || o.v == 0 && inI(i100, 11, 14):
		return "many"
	}
	return "other"
}

func pluralRulesFor(locale string) (func(pluralOperands) string, error) {
	pl, ok := locales().Plural[locale]
	if !ok {
		return nil, &Unsupported{What: "plural rules for " + locale}
	}
	if r, ok := pluralRuleSets[pl]; ok {
		return r, nil
	}
	lang, _, _ := strings.Cut(pl, "-")
	if r, ok := pluralRuleSets[lang]; ok {
		return r, nil
	}
	return nil, &Unsupported{What: "plural rules " + pl}
}

// PluralSelect mirrors new Intl.PluralRules(locale).select(v).
func PluralSelect(locale string, v float64) (string, error) {
	r, err := pluralRulesFor(locale)
	if err != nil {
		return "", err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "other", nil
	}
	ip, fp := roundDecimal(v, 3)
	return r(operandsOf(ip, fp)), nil
}
