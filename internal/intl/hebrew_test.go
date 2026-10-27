package intl

import (
	"strconv"
	"testing"
)

func TestHebrewNumeralYears(t *testing.T) {
	var years map[string]string
	readTestdata(t, "hebrewyears", &years)
	bad := 0
	for ys, want := range years {
		y, _ := strconv.ParseInt(ys, 10, 64)
		if got := hebrewNumeral(y); got != want {
			if bad < 10 {
				t.Errorf("year %d: got %q want %q", y, got, want)
			}
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%d of %d years differ", bad, len(years))
	}
}
