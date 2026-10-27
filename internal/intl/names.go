package intl

import (
	"math"
	"sort"
	"sync"
)

type namesZone struct {
	Periods [][2]int64 `json:"periods"`
	Far     [][2]any   `json:"far"`
}

type namesData struct {
	Contexts []string              `json:"contexts"`
	Strings  []string              `json:"strings"`
	Vectors  [][]int               `json:"vectors"`
	Zones    map[string]*namesZone `json:"zones"`
}

var (
	namesOnce  sync.Once
	names      namesData
	namesCtx   map[string]int
	namesLower map[string]string
)

const (
	namesEnd   = 4133980800 // 2101-01-01T00:00:00Z, the end of the scanned range
	namesSlots = 4
)

func namedata() *namesData {
	namesOnce.Do(func() {
		loadJSON("names", &names)
		namesCtx = make(map[string]int, len(names.Contexts))
		for i, c := range names.Contexts {
			namesCtx[c] = i
		}
		namesLower = make(map[string]string, len(names.Zones))
		for id := range names.Zones {
			namesLower[asciiLower(id)] = id
		}
	})
	return &names
}

func namesFor(zone string) *namesZone {
	d := namedata()
	if z, ok := d.Zones[zone]; ok {
		return z
	}
	if id, ok := namesLower[asciiLower(zone)]; ok {
		return d.Zones[id]
	}
	return nil
}

// ZoneName returns the name Luxon's parseZoneInfo bag (timeZoneName "short" or "long" with
// hourCycle h23 and two-digit numeric fields) gets from ICU. The bool is false when ICU shows no
// name part.
func ZoneName(locale, zone string, ms float64, style string) (string, bool, error) {
	switch style {
	case "short":
		return zoneNameSlot(locale, zone, ms, 0)
	case "long":
		return zoneNameSlot(locale, zone, ms, 1)
	}
	if _, err := ResolveDateTimeFormat(locale); err != nil {
		return "", false, err
	}
	return "", false, &Unsupported{What: "timeZoneName " + style}
}

// zoneNameSlot reads one recorded name slot: 0 and 1 for Luxon's bag, 2 and 3 for the plain
// short and long name fields.
func zoneNameSlot(locale, zone string, ms float64, slot int) (string, bool, error) {
	if !validTime(ms) {
		return "", false, knownError("dtfNaN")
	}
	d := namedata()
	if locale == "" {
		locale = SystemLocale()
	}
	ci, ok := namesCtx[locale]
	if !ok {
		if _, err := ResolveDateTimeFormat(locale); err != nil {
			return "", false, err
		}
		return "", false, &Unsupported{What: "zone names for locale " + locale}
	}
	if zone == "" {
		zone = SystemTimeZone()
	}
	canon, ok := CanonicalTimeZone(zone)
	if !ok {
		return "", false, rangeError("Invalid time zone specified: " + zone)
	}
	z := namesFor(zone)
	if z == nil {
		return "", false, &Unsupported{What: "zone names for " + zone}
	}
	vec := z.vectorAt(canon, int64(math.Floor(ms/1000)))
	s := d.Vectors[vec][namesSlots*ci+slot]
	if s < 0 {
		return "", false, nil
	}
	return d.Strings[s], true, nil
}

func (z *namesZone) vectorAt(canon string, sec int64) int {
	p := z.Periods
	if sec < p[0][0] {
		return int(p[0][1])
	}
	if sec <= namesEnd {
		i := sort.Search(len(p), func(i int) bool { return p[i][0] > sec }) - 1
		return int(p[i][1])
	}
	off := OffsetSeconds(canon, sec)
	if sec >= metazoneEnd {
		for _, f := range z.Far {
			ms, v := f[0].(float64), f[1]
			if vec, ok := v.(float64); ok && ms/1000 >= metazoneEnd && OffsetSeconds(canon, int64(math.Floor(ms/1000))) == off {
				return int(vec)
			}
		}
	}
	for i := len(p) - 1; i >= 0; i-- {
		if OffsetSeconds(canon, p[i][0]) == off {
			return int(p[i][1])
		}
	}
	return int(p[len(p)-1][1])
}

// metazoneEnd is ICU's default end of a metazone mapping, 9999-12-31 23:59 UTC; later instants
// get GMT-format names wherever the name came from a metazone.
const metazoneEnd = 253402300740
