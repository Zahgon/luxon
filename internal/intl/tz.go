package intl

import (
	"sort"
	"strconv"
	"sync"
)

type tzZone struct {
	First int64      `json:"first"`
	Trans [][2]int64 `json:"trans"`
	Rule  [][5]int64 `json:"rule"`
	Cycle [][2]int64 `json:"cycle"`
}

type tzData struct {
	IDs   map[string]string  `json:"ids"`
	Zones map[string]*tzZone `json:"zones"`
	Y2100 int64              `json:"y2100"`
}

var (
	tzOnce sync.Once
	tzd    tzData
)

func tzdata() *tzData {
	tzOnce.Do(func() { loadJSON("tz", &tzd) })
	return &tzd
}

const cycleSeconds = 146097 * 86400

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// parseOffsetID accepts the UTC offset time zone forms ±HH, ±HHMM and ±HH:MM.
func parseOffsetID(s string) (int64, bool) {
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') {
		return 0, false
	}
	digits := func(t string) (int64, bool) {
		if len(t) != 2 || t[0] < '0' || t[0] > '9' || t[1] < '0' || t[1] > '9' {
			return 0, false
		}
		return int64(t[0]-'0')*10 + int64(t[1]-'0'), true
	}
	rest := s[1:]
	var hh, mm int64
	var ok bool
	switch len(rest) {
	case 2:
		hh, ok = digits(rest)
	case 4:
		if hh, ok = digits(rest[:2]); ok {
			mm, ok = digits(rest[2:])
		}
	case 5:
		if rest[2] != ':' {
			return 0, false
		}
		if hh, ok = digits(rest[:2]); ok {
			mm, ok = digits(rest[3:])
		}
	}
	if !ok || hh > 23 || mm > 59 {
		return 0, false
	}
	off := (hh*60 + mm) * 60
	if s[0] == '-' {
		off = -off
	}
	return off, true
}

func formatOffsetID(off int64) string {
	sign := "+"
	if off < 0 {
		sign = "-"
		off = -off
	}
	h, m := off/3600, off/60%60
	two := func(n int64) string {
		if n < 10 {
			return "0" + strconv.FormatInt(n, 10)
		}
		return strconv.FormatInt(n, 10)
	}
	return sign + two(h) + ":" + two(m)
}

// CanonicalTimeZone returns the identifier that resolvedOptions().timeZone reports for zone, and
// false when Intl.DateTimeFormat rejects it.
func CanonicalTimeZone(zone string) (string, bool) {
	if c, ok := tzdata().IDs[asciiLower(zone)]; ok {
		return c, true
	}
	if off, ok := parseOffsetID(zone); ok {
		return formatOffsetID(off), true
	}
	return "", false
}

// OffsetSeconds returns the UTC offset, in seconds, of a canonical zone at the given epoch second.
func OffsetSeconds(canonical string, sec int64) int64 {
	if off, ok := parseOffsetID(canonical); ok {
		return off
	}
	z := tzdata().Zones[canonical]
	if z == nil {
		return 0
	}
	return z.offsetAt(sec)
}

func (z *tzZone) baseAt2100() int64 {
	if len(z.Trans) == 0 {
		return z.First
	}
	return z.Trans[len(z.Trans)-1][1]
}

func (z *tzZone) offsetAt(sec int64) int64 {
	y2100 := tzdata().Y2100
	if sec < y2100 || (z.Rule == nil && z.Cycle == nil) {
		i := sort.Search(len(z.Trans), func(i int) bool { return z.Trans[i][0] > sec })
		if i == 0 {
			return z.First
		}
		return z.Trans[i-1][1]
	}
	base := z.baseAt2100()
	if z.Rule != nil {
		y, _, _ := civilFromDays(floorDiv(sec+base, 86400))
		off := base
		found := false
		for yy := y - 1; yy <= y+1; yy++ {
			prev := z.Rule[len(z.Rule)-1][4]
			for _, t := range ruleTransitions(z.Rule, yy, prev) {
				if t[0] >= y2100 && t[0] <= sec {
					off = t[1]
					found = true
				}
			}
		}
		if !found {
			return base
		}
		return off
	}
	k := floorDiv(sec-y2100, cycleSeconds)
	s := sec - k*cycleSeconds
	i := sort.Search(len(z.Cycle), func(i int) bool { return z.Cycle[i][0] > s })
	if i == 0 {
		if k == 0 || len(z.Cycle) == 0 {
			return base
		}
		return z.Cycle[len(z.Cycle)-1][1]
	}
	return z.Cycle[i-1][1]
}

// ruleTransitions returns the [epochSecond, newOffset] transitions of a yearly rule. Each rule row
// is [month, day, weekday (0=Sunday, -1 for an exact day), wall seconds in the old offset, offset].
func ruleTransitions(rule [][5]int64, year, prev int64) [][2]int64 {
	out := make([][2]int64, 0, len(rule))
	off := prev
	for _, r := range rule {
		days := daysFromCivil(year, r[0], r[1])
		if r[2] >= 0 {
			cur := floorMod(days+4, 7)
			days += floorMod(r[2]-cur, 7)
		}
		out = append(out, [2]int64{days*86400 + r[3] - off, r[4]})
		off = r[4]
	}
	return out
}

// daysFromCivil converts a proleptic Gregorian date to days since 1970-01-01.
func daysFromCivil(y, m, d int64) int64 {
	if m <= 2 {
		y--
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	mp := m + 9
	if m > 2 {
		mp = m - 3
	}
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// civilFromDays converts days since 1970-01-01 to a proleptic Gregorian date.
func civilFromDays(z int64) (y, m, d int64) {
	z += 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y = yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d = doy - (153*mp+2)/5 + 1
	if mp < 10 {
		m = mp + 3
	} else {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return
}
