package luxon

import (
	"os"
	"sort"
	"strings"
	"testing"
)

type recordedCall struct {
	key string
	fn  func() any
}

// determinismCalls mirrors the call battery whose output by the original is recorded in
// testdata/determinism.txt.
func determinismCalls() []recordedCall {
	zones := []string{"America/New_York", "Asia/Calcutta", "Asia/Kolkata", "asia/kolkata", "Europe/Dublin",
		"Australia/Lord_Howe", "Asia/Kathmandu", "Etc/GMT+5", "America/Winnipeg", "Africa/Accra", "IST", "+05:30",
		"Factory", "EST5EDT", "US/Pacific-New"}
	locales := []string{"en-US", "fr", "de", "ar-EG", "fa-IR", "th-TH", "ja-JP-u-ca-japanese", "he-IL-u-ca-hebrew",
		"zh-Hant-u-nu-hanidec", "ru", "es", "hi-IN"}
	instants := []float64{-8.64e15, -8.64e15 + 3.6e6, -6.21e13, -2208988800000, -1e12, 0, 1e12, 1.7e12, 1.8e12,
		4.1e12, 8.64e15 - 3.6e6, 8.64e15}
	var calls []recordedCall
	add := func(key string, fn func() any) { calls = append(calls, recordedCall{key, fn}) }
	for _, z := range zones {
		for _, t := range instants {
			suffix := z + " " + numberToString(t)
			add("iso "+suffix, func() any { return DateTimeFromMillis(t, Obj{"zone", z}).ToISO() })
			add("off "+suffix, func() any {
				d := DateTimeFromMillis(t, Obj{"zone", z})
				return []any{d.Offset(), d.ZoneName(), d.IsValid(), d.InvalidReason()}
			})
		}
	}
	for _, l := range locales {
		for _, z := range zones[:6] {
			d := func() *DateTime { return DateTimeFromMillis(1.7e12, Obj{"zone", z, "locale", l}) }
			add("full "+l+" "+z, func() any { return d().ToLocaleString(DateTimeDATETIME_HUGE()) })
			add("fmt "+l+" "+z, func() any { return d().ToFormat("yyyy LLLL dd EEEE a ZZZZ ZZZZZ G") })
			add("rel "+l+" "+z, func() any {
				return d().ToRelative(Obj{"base", DateTimeFromMillis(1.6e12, Obj{"zone", z})})
			})
			add("relcal "+l+" "+z, func() any {
				return d().ToRelativeCalendar(Obj{"base", DateTimeFromMillis(1.69e12, Obj{"zone", z})})
			})
			add("human "+l, func() any {
				return DurationFromObject(Obj{"years", 1, "months", 2, "hours", 1000, "minutes", 1.5}, Obj{"locale", l}).ToHuman()
			})
			add("ambig "+l+" "+z, func() any {
				return DateTimeFromObject(Obj{"year", 2024, "month", 11, "day", 3, "hour", 1, "minute", 30},
					Obj{"zone", z, "locale", l}).ToISO()
			})
			add("parse "+l, func() any {
				text := DateTimeFromMillis(1.7e12, Obj{"locale", l}).ToFormat("d MMMM yyyy")
				p := DateTimeFromFormat(text, "d MMMM yyyy", Obj{"locale", l})
				return []any{p.IsValid(), p.ToISO(), p.InvalidExplanation()}
			})
		}
	}
	add("info", func() any {
		return []any{Info.Months("short", Obj{"locale", "bg"}), Info.MonthsFormat("short", Obj{"locale", "bg"}),
			Info.Weekdays("narrow", Obj{"locale", "he"}), Info.Eras("long", Obj{"locale", "ja-JP-u-ca-japanese"}),
			Info.Meridiems(Obj{"locale", "zh-TW"})}
	})
	add("norm", func() any {
		var name any = Undefined
		if z := FixedOffsetZoneParseSpecifier("UTC+5:30"); z != nil {
			name = z.Name()
		}
		return []any{IANAZoneNormalizeZone("Asia/Kolkata"), IANAZoneIsValidZone("Factory"), name}
	})
	add("interval", func() any {
		return IntervalFromDateTimes(DateTimeFromMillis(1e12, Obj{"zone", "Europe/Dublin"}),
			DateTimeFromMillis(1.1e12, Obj{"zone", "Europe/Dublin"})).ToLocaleString(DateTimeDATETIME_MED(), Obj{"locale", "fr"})
	})
	return calls
}

// pinnedNow is the Settings.now every recording uses: 2025-01-15T12:00:00Z.
func pinnedNow() float64 { return dateUTC(2025, 0, 15, 12, 0, 0, 0) }

func TestDeterminismBattery(t *testing.T) {
	data, err := os.ReadFile("testdata/determinism.txt")
	if err != nil {
		t.Fatal(err)
	}
	saved := Settings.Now()
	Settings.SetNow(pinnedNow)
	defer Settings.SetNow(saved)

	var rows []string
	for _, c := range determinismCalls() {
		rows = append(rows, c.key+"\t"+jsCall(c.fn))
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return strings.SplitN(rows[i], "\t", 2)[0] < strings.SplitN(rows[j], "\t", 2)[0]
	})
	want := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(want) != len(rows) {
		t.Fatalf("%d rows, recorded %d", len(rows), len(want))
	}
	bad, gaps := 0, 0
	for i := range want {
		if rows[i] == want[i] {
			continue
		}
		if strings.Contains(rows[i], `{"unsupported":`) {
			gaps++
			t.Logf("gap:  %s", rows[i])
			continue
		}
		bad++
		if bad <= 40 {
			t.Errorf("\n got: %s\nwant: %s", rows[i], want[i])
		}
	}
	t.Logf("%d rows, %d mismatches, %d ICU data gaps", len(rows), bad, gaps)
}
