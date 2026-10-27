package intl

import (
	"math"
	"testing"
)

type tzSample struct {
	First       float64       `json:"first"`
	Trans       [][2]float64  `json:"trans"`
	Far         []*float64    `json:"far"`
	After       [][2]*float64 `json:"after"`
	Before      [][2]*float64 `json:"before"`
	AtMinus3000 *float64      `json:"atMinus3000"`
	PeriodicBad [][]any       `json:"periodicBad"`
}

var farInstants = []float64{-8.64e15, -8.64e15 + 1000, -6e15, -1e15, -1e14, -6e13, -1e13, -7e12, 5e12, 1e13, 4.2e12, 6e12, 1e14, 1e15, 6e15, 8.64e15 - 1000, 8.64e15}

// luxonOffset mirrors what IANAZone.offset reports: minutes, NaN when the wall time leaves the
// ECMAScript time range.
func luxonOffset(zone string, ms float64) float64 {
	sec := int64(math.Floor(ms / 1000))
	off := OffsetSeconds(zone, sec)
	if !validTime(float64(sec*1000 + off*1000)) {
		return math.NaN()
	}
	return float64(off) / 60
}

func sameOffset(a float64, b *float64) bool {
	if b == nil {
		return math.IsNaN(a)
	}
	return math.Abs(a-*b) < 1e-9
}

func TestOffsets(t *testing.T) {
	var zones map[string]tzSample
	readTestdata(t, "tz", &zones)
	start := float64(-5364662400) // 1800-01-01T00:00:00Z
	checked := 0
	for id, z := range zones {
		canon, ok := CanonicalTimeZone(id)
		if !ok {
			t.Errorf("%s: not recognised", id)
			continue
		}
		at := func(sec float64) float64 { return luxonOffset(canon, sec*1000) }
		first := z.First
		if got := at(start); !sameOffset(got, &first) {
			t.Errorf("%s: offset at 1800 = %v, want %v", id, got, first)
		}
		prev := z.First
		for _, tr := range append(append([][2]float64{}, z.Trans...), floatPairs(z.After)...) {
			if got := at(tr[0] - 1); !sameOffset(got, &prev) {
				t.Errorf("%s: offset before %v = %v, want %v", id, tr[0], got, prev)
			}
			want := tr[1]
			if got := at(tr[0]); !sameOffset(got, &want) {
				t.Errorf("%s: offset at %v = %v, want %v", id, tr[0], got, want)
			}
			prev = tr[1]
			checked++
		}
		for i, ms := range farInstants {
			if got := luxonOffset(canon, ms); !sameOffset(got, z.Far[i]) {
				t.Errorf("%s: offset at %v = %v, want %v", id, ms, got, z.Far[i])
			}
		}
		if len(z.Before) != 0 || len(z.PeriodicBad) != 0 {
			t.Errorf("%s: extraction reported transitions before 1800 or a broken 400-year period", id)
		}
	}
	if checked < 10000 {
		t.Errorf("only %d transitions checked", checked)
	}
}

func floatPairs(in [][2]*float64) [][2]float64 {
	out := make([][2]float64, 0, len(in))
	for _, p := range in {
		if p[0] != nil && p[1] != nil {
			out = append(out, [2]float64{*p[0], *p[1]})
		}
	}
	return out
}
