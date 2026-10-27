package intl

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type dtfVectors struct {
	Parts  [][]json.RawMessage `json:"parts"`
	Ranges [][]json.RawMessage `json:"ranges"`
}

func bagFromKey(t *testing.T, key string) map[string]any {
	var pairs [][2]any
	if err := json.Unmarshal([]byte(key), &pairs); err != nil {
		t.Fatal(err)
	}
	m := map[string]any{}
	for _, p := range pairs {
		m[p[0].(string)] = p[1]
	}
	return m
}

func expectedError(raw json.RawMessage) (string, bool) {
	var e []string
	if json.Unmarshal(raw, &e) == nil && len(e) == 2 && len(e[0]) > 0 && e[0][0] == '!' {
		return e[0][1:] + ": " + e[1], true
	}
	return "", false
}

func unsupportedKind(u *Unsupported) string {
	w := strings.Fields(u.What)
	if len(w) > 2 {
		w = w[:2]
	}
	return strings.Join(w, " ")
}

func TestDateTimeFormatVectors(t *testing.T) {
	var v dtfVectors
	readTestdata(t, "dtf", &v)
	bad, unsupported := 0, 0
	why := map[string]int{}
	report := func(format string, args ...any) {
		if bad < 15 {
			t.Errorf(format, args...)
		}
		bad++
	}
	for _, r := range v.Parts {
		var ctx, key, tz string
		var ms float64
		json.Unmarshal(r[0], &ctx)
		json.Unmarshal(r[1], &key)
		json.Unmarshal(r[2], &tz)
		json.Unmarshal(r[3], &ms)
		opts := bagFromKey(t, key)
		opts["timeZone"] = tz
		f, err := NewDateTimeFormat(ctx, opts)
		var got []Part
		if err == nil {
			got, err = f.FormatToParts(ms)
		}
		var u *Unsupported
		if errors.As(err, &u) {
			unsupported++
			why[unsupportedKind(u)]++
			continue
		}
		if want, isErr := expectedError(r[4]); isErr {
			if err == nil || err.Error() != want {
				report("%s %s %s %v: got %v, want error %s", ctx, key, tz, ms, got, want)
			}
			continue
		}
		var want [][2]string
		json.Unmarshal(r[4], &want)
		if err != nil {
			report("%s %s %s %v: %v", ctx, key, tz, ms, err)
			continue
		}
		gotPairs := make([][2]string, len(got))
		for i, p := range got {
			gotPairs[i] = [2]string{p.Type, p.Value}
		}
		if fmt.Sprint(gotPairs) != fmt.Sprint(want) {
			report("%s %s %s %v:\n got %q\nwant %q", ctx, key, tz, ms, gotPairs, want)
		}
	}
	for _, r := range v.Ranges {
		var ctx, key, tz string
		var a, b float64
		json.Unmarshal(r[0], &ctx)
		json.Unmarshal(r[1], &key)
		json.Unmarshal(r[2], &tz)
		json.Unmarshal(r[3], &a)
		json.Unmarshal(r[4], &b)
		opts := bagFromKey(t, key)
		opts["timeZone"] = tz
		f, err := NewDateTimeFormat(ctx, opts)
		var got string
		if err == nil {
			got, err = f.FormatRange(a, b)
		}
		var u *Unsupported
		if errors.As(err, &u) {
			unsupported++
			why[unsupportedKind(u)]++
			continue
		}
		if want, isErr := expectedError(r[5]); isErr {
			if err == nil || err.Error() != want {
				report("range %s %s %s %v %v: got %q, want error %s", ctx, key, tz, a, b, got, want)
			}
			continue
		}
		var want string
		json.Unmarshal(r[5], &want)
		if err != nil {
			report("range %s %s %s %v %v: %v", ctx, key, tz, a, b, err)
			continue
		}
		if got != want {
			report("range %s %s %s %v %v:\n got %q\nwant %q", ctx, key, tz, a, b, got, want)
		}
	}
	t.Logf("%d parts vectors, %d range vectors, %d unsupported %v", len(v.Parts), len(v.Ranges), unsupported, why)
	if bad > 0 {
		t.Errorf("%d mismatches", bad)
	}
}
