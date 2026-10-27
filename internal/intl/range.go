package intl

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync"
)

// rangeTemplate is one formatRange level: its part shape and, per "type|source", the values ICU
// renders there (interval patterns have their own field widths and zone-name fields).
type rangeTemplate struct {
	Shape [][]string        `json:"shape"`
	Lists map[string]int    `json:"lists"`
	Years map[string]string `json:"years"`
	Minus string            `json:"minus"`
	TZ    int               `json:"tz"`
}

type rangeData struct {
	Lists     []map[string]string                              `json:"lists"`
	Shapes    [][][]string                                     `json:"shapes"`
	Templates []*rangeTemplate                                 `json:"templates"`
	Contexts  map[string]map[string]map[string]json.RawMessage `json:"contexts"`
}

var (
	rangeOnce sync.Once
	rangeDat  rangeData
)

func rangedata() *rangeData {
	rangeOnce.Do(func() { loadJSON("ranges", &rangeDat) })
	return &rangeDat
}

func rangeLevel(a, b instantFields) string {
	switch {
	case a.cal.Era != b.cal.Era:
		return "G"
	case a.cal.Year != b.cal.Year:
		return "y"
	case a.cal.Month != b.cal.Month:
		return "M"
	case a.cal.Day != b.cal.Day:
		return "d"
	case (a.hour < 12) != (b.hour < 12):
		return "a"
	case a.hour != b.hour:
		return "h"
	case a.minute != b.minute:
		return "m"
	case a.second != b.second:
		return "s"
	case a.milli != b.milli:
		return "ms"
	}
	return "same"
}

// rangeSlots are the keys a range part's values may be recorded under, most specific first: the
// part's own type, source and occurrence within the shape, then the same type elsewhere.
func rangeSlots(typ, src string, n int) []string {
	keys := []string{typ + "|" + src + "|" + strconv.Itoa(n)}
	for _, s := range []string{src, "shared", "startRange", "endRange"} {
		for i := 0; i < 3; i++ {
			keys = append(keys, typ+"|"+s+"|"+strconv.Itoa(i))
		}
	}
	return keys
}

type mergeKey struct{ r, s int }

var mergedLists sync.Map

// completeList fills the values a range slot's sample pairs never reached from the single-date
// template, provided the two agree on at least one value and differ on none (the interval
// pattern renders that field the same way).
func completeList(ri int, typ string, tpl *dtfTemplate) map[string]string {
	rl := rangedata().Lists[ri]
	si, ok := tpl.Lists[typ]
	if !ok {
		return rl
	}
	k := mergeKey{ri, si}
	if m, ok := mergedLists.Load(k); ok {
		return m.(map[string]string)
	}
	sl := dtfdata().Lists[si]
	agree := false
	for key, v := range rl {
		if sv, ok := sl[key]; ok {
			if sv != v {
				mergedLists.Store(k, rl)
				return rl
			}
			agree = true
		}
	}
	if !agree {
		mergedLists.Store(k, rl)
		return rl
	}
	m := make(map[string]string, len(sl)+len(rl))
	for key, v := range sl {
		m[key] = v
	}
	for key, v := range rl {
		m[key] = v
	}
	mergedLists.Store(k, m)
	return m
}

func (f *DateTimeFormat) rangeData(rt *rangeTemplate, src string, n int) partData {
	list := func(typ string) map[string]string {
		for _, k := range rangeSlots(typ, src, n) {
			if i, ok := rt.Lists[k]; ok {
				return completeList(i, typ, f.tpl)
			}
		}
		return f.list(typ)
	}
	year := f.tpl.Year
	for _, k := range rangeSlots("year", src, n) {
		if y, ok := rt.Years[k]; ok {
			year = y
			break
		}
	}
	minus := rt.Minus
	if minus == "" {
		minus = f.tpl.Minus
	}
	return partData{list: list, year: year, minus: minus, tz: rt.TZ}
}

// FormatRangeToParts mirrors Intl.DateTimeFormat.prototype.formatRangeToParts; the second
// result is each part's source (startRange, endRange or shared).
func (f *DateTimeFormat) FormatRangeToParts(start, end float64) ([]Part, []string, error) {
	if err := checkTime(start); err != nil {
		return nil, nil, err
	}
	if err := checkTime(end); err != nil {
		return nil, nil, err
	}
	start, end = math.Trunc(start), math.Trunc(end)
	a, b := f.fieldsAt(start), f.fieldsAt(end)
	level := rangeLevel(a, b)
	levels := rangedata().Contexts[f.locale][f.bagKey]
	if levels == nil {
		return nil, nil, &Unsupported{What: "date range format " + f.bagKey + " in " + f.locale}
	}
	raw, ok := levels[level]
	if !ok && level == "G" {
		level = "y"
		raw, ok = levels[level]
	}
	if !ok {
		return nil, nil, &Unsupported{What: "date range level " + level + " in " + f.locale}
	}
	var entry struct {
		R         *int     `json:"r"`
		Collapsed bool     `json:"collapsed"`
		Throw     []string `json:"throw"`
	}
	var shapeIdx int
	var rt *rangeTemplate
	if err := json.Unmarshal(raw, &shapeIdx); err != nil {
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, nil, &Unsupported{What: "date range level " + level}
		}
		switch {
		case len(entry.Throw) == 2:
			return nil, nil, (throwData{Throw: entry.Throw}).err()
		case entry.Collapsed:
			parts, err := f.FormatToParts(start)
			sources := make([]string, len(parts))
			for i := range sources {
				sources[i] = "shared"
			}
			return parts, sources, err
		case entry.R != nil:
			rt = rangedata().Templates[*entry.R]
		default:
			return nil, nil, &Unsupported{What: "date range level " + level}
		}
	}
	var shape [][]string
	if rt != nil {
		shape = rt.Shape
	} else {
		shape = rangedata().Shapes[shapeIdx]
	}
	firstEnd := len(shape)
	for i, p := range shape {
		if len(p) > 1 && p[1] == "endRange" {
			firstEnd = i
			break
		}
	}
	occ := make([]int, len(shape))
	seen := map[string]int{}
	for i, p := range shape {
		k := p[0] + "|" + p[1]
		occ[i] = seen[k]
		seen[k]++
	}
	single := f.single()
	return f.render(shape, func(i int, src string) (instantFields, partData) {
		x := a
		if src == "endRange" || (src == "shared" && i > firstEnd) {
			x = b
		}
		if rt == nil {
			return x, single
		}
		return x, f.rangeData(rt, src, occ[i])
	})
}

// FormatRange mirrors Intl.DateTimeFormat.prototype.formatRange: a range that collapses to one
// date (every part shared) is formatted by format(start).
func (f *DateTimeFormat) FormatRange(start, end float64) (string, error) {
	parts, sources, err := f.FormatRangeToParts(start, end)
	if err != nil {
		return "", err
	}
	for _, s := range sources {
		if s != "shared" {
			return joinParts(parts), nil
		}
	}
	return strings.ReplaceAll(joinParts(parts), "\u202f", " "), nil
}
