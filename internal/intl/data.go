// Package intl reproduces the parts of the ECMAScript Intl API (as implemented by V8 with the ICU
// and CLDR data compiled into Node.js 24) that Luxon depends on: time zone offsets and names,
// DateTimeFormat patterns, NumberFormat, PluralRules, RelativeTimeFormat, ListFormat and locale
// week data. The data under data/ was generated from that runtime.
package intl

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"math"
)

//go:embed data/*.json.gz
var dataFS embed.FS

func loadJSON(name string, v any) {
	f, err := dataFS.Open("data/" + name + ".json.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(zr).Decode(v); err != nil {
		panic(err)
	}
}

// Error is a JavaScript exception raised by an Intl constructor or method.
type Error struct {
	Name    string
	Message string
}

func (e *Error) Error() string { return e.Name + ": " + e.Message }

func rangeError(msg string) *Error { return &Error{Name: "RangeError", Message: msg} }

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 { return a - floorDiv(a, b)*b }

const maxTime = 8.64e15

// validTime reports whether ms is a valid ECMAScript time value.
func validTime(ms float64) bool { return !math.IsNaN(ms) && math.Abs(ms) <= maxTime }
