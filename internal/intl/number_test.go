package intl

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

type miscSamples struct {
	Inputs struct {
		NUMS            []string   `json:"NUMS"`
		GROUPED         []float64  `json:"GROUPED"`
		UNIT_VALS       []string   `json:"UNIT_VALS"`
		REL_VALS        []string   `json:"REL_VALS"`
		PLURAL_VALS     []float64  `json:"PLURAL_VALS"`
		LIST_INPUTS     [][]string `json:"LIST_INPUTS"`
		LIST_CONTEXTUAL [][]string `json:"LIST_CONTEXTUAL"`
		UNITS           []string   `json:"UNITS"`
		REL_UNITS       []string   `json:"REL_UNITS"`
		SIGNS           []string   `json:"SIGNS"`
	} `json:"inputs"`
	Contexts map[string]struct {
		Numbers map[string][]string `json:"numbers"`
		Grouped []string            `json:"grouped"`
		Units   map[string][]string `json:"units"`
		Rel     map[string][]string `json:"rel"`
		Lists   map[string]struct {
			Basic []string `json:"basic"`
			Ctx   []string `json:"ctx"`
		} `json:"lists"`
		Plural struct {
			Values []string `json:"values"`
		} `json:"plural"`
	} `json:"contexts"`
}

func jsNumber(s string) float64 {
	switch s {
	case "-0":
		return math.Copysign(0, -1)
	case "NaN":
		return math.NaN()
	case "Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(s)
	}
	return v
}

type mismatchCounter struct {
	t    *testing.T
	area string
	bad  int
}

func (m *mismatchCounter) check(ctx, what, got, want string) {
	if got != want {
		if m.bad < 8 {
			m.t.Errorf("%s %s %s: got %q want %q", m.area, ctx, what, got, want)
		}
		m.bad++
	}
}

func TestNumbersUnitsRelativeListsPlurals(t *testing.T) {
	var s miscSamples
	readTestdata(t, "misc", &s)
	nums := &mismatchCounter{t: t, area: "number"}
	units := &mismatchCounter{t: t, area: "unit"}
	rels := &mismatchCounter{t: t, area: "relative"}
	lists := &mismatchCounter{t: t, area: "list"}
	plurals := &mismatchCounter{t: t, area: "plural"}
	for ctx, c := range s.Contexts {
		for key, outs := range c.Numbers {
			midS, sd, _ := strings.Cut(key, "|")
			mid, _ := strconv.Atoi(midS)
			f, err := NewNumberFormat(ctx, NumberOptions{MinimumIntegerDigits: mid, SignDisplay: sd})
			if err != nil {
				t.Fatalf("%s: %v", ctx, err)
			}
			for i, in := range s.Inputs.NUMS {
				v := jsNumber(in)
				if i == 1 {
					v, in = math.Copysign(0, -1), "-0"
				}
				nums.check(ctx, key+" "+in, f.Format(v), outs[i])
			}
		}
		g, _ := NewNumberFormat(ctx, NumberOptions{UseGrouping: true})
		for i, v := range s.Inputs.GROUPED {
			nums.check(ctx, "grouped "+strconv.FormatFloat(v, 'g', -1, 64), g.Format(v), c.Grouped[i])
		}
		for key, outs := range c.Units {
			disp, unit, _ := strings.Cut(key, "|")
			f, err := NewNumberFormat(ctx, NumberOptions{Unit: unit, UnitDisplay: disp})
			if err != nil {
				t.Fatalf("%s: %v", ctx, err)
			}
			for i, in := range s.Inputs.UNIT_VALS {
				units.check(ctx, key+" "+in, f.Format(jsNumber(in)), outs[i])
			}
		}
		for key, outs := range c.Rel {
			parts := strings.Split(key, "|")
			f, err := NewRelativeTimeFormat(ctx, parts[0], parts[1])
			if err != nil {
				t.Fatalf("%s: %v", ctx, err)
			}
			for i, in := range s.Inputs.REL_VALS {
				got, err := f.Format(jsNumber(in), parts[2])
				if err != nil {
					got = err.Error()
				}
				rels.check(ctx, key+" "+in, got, outs[i])
			}
		}
		for key, x := range c.Lists {
			typ, style, _ := strings.Cut(key, "|")
			f, err := NewListFormat(ctx, typ, style)
			if err != nil {
				t.Fatalf("%s: %v", ctx, err)
			}
			for i, in := range s.Inputs.LIST_INPUTS {
				lists.check(ctx, key+" "+strings.Join(in, ","), f.Format(in), x.Basic[i])
			}
			for i, in := range s.Inputs.LIST_CONTEXTUAL {
				lists.check(ctx, key+" "+strings.Join(in, ","), f.Format(in), x.Ctx[i])
			}
		}
		for i, v := range s.Inputs.PLURAL_VALS {
			got, err := PluralSelect(ctx, v)
			if err != nil {
				t.Fatalf("%s: %v", ctx, err)
			}
			plurals.check(ctx, strconv.FormatFloat(v, 'g', -1, 64), got, c.Plural.Values[i])
		}
	}
	for _, m := range []*mismatchCounter{nums, units, rels, lists, plurals} {
		if m.bad > 0 {
			t.Errorf("%s: %d mismatches", m.area, m.bad)
		}
	}
}
