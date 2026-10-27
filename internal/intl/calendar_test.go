package intl

import (
	"compress/gzip"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

func readTestdata(t *testing.T, name string, v any) {
	t.Helper()
	f, err := os.Open("testdata/" + name + ".json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(zr).Decode(v); err != nil {
		t.Fatal(err)
	}
}

// TestCalendarFields checks every calendar against the fields ICU reports for the same instants.
func TestCalendarFields(t *testing.T) {
	var samples map[string][][]any
	readTestdata(t, "calendar", &samples)
	for calendar, rows := range samples {
		bad := 0
		for _, r := range rows {
			ms := r[0].(float64)
			if len(r) == 2 {
				continue
			}
			day := int64(math.Floor(ms / 86400000))
			got := calendarFields(calendar, day)
			era, _ := r[1].(string)
			year, _ := strconv.ParseInt(r[2].(string), 10, 64)
			month := r[3].(string)
			dom, _ := strconv.ParseInt(r[4].(string), 10, 64)
			gotMonth := strings.SplitN(got.Month, "|", 2)[0]
			gotEra := got.Era
			if calendar == "iso8601" {
				gotEra = ""
			}
			if gotEra != era || got.Year != year || gotMonth != month || got.Day != dom {
				if bad < 5 {
					t.Errorf("%s at %v: got %+v want era=%q year=%d month=%q day=%d", calendar, ms, got, era, year, month, dom)
				}
				bad++
			}
		}
		if bad > 0 {
			t.Errorf("%s: %d of %d samples differ", calendar, bad, len(rows))
		}
	}
}
