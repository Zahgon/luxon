package intl

import (
	"testing"
)

// TestZoneNames checks every recorded name period and the far instants of every recorded zone.
func TestZoneNames(t *testing.T) {
	d := namedata()
	bad := 0
	report := func(format string, args ...any) {
		if bad < 10 {
			t.Errorf(format, args...)
		}
		bad++
	}
	check := func(zone string, ms float64, vec int) {
		for ci, ctx := range d.Contexts {
			for slot := 0; slot < namesSlots; slot++ {
				style := slot
				want := d.Vectors[vec][namesSlots*ci+slot]
				got, ok, err := zoneNameSlot(ctx, zone, ms, slot)
				if err != nil {
					report("%s %s %v slot %d: %v", zone, ctx, ms, style, err)
					return
				}
				if (want < 0) != !ok || (ok && got != d.Strings[want]) {
					w := "<none>"
					if want >= 0 {
						w = d.Strings[want]
					}
					report("%s %s %v slot %d: got %q want %q", zone, ctx, ms, style, got, w)
				}
			}
		}
	}
	for zone, z := range d.Zones {
		for _, p := range z.Periods {
			check(zone, float64(p[0])*1000, int(p[1]))
		}
		for _, f := range z.Far {
			ms := f[0].(float64)
			if v, ok := f[1].(float64); ok {
				check(zone, ms, int(v))
			} else if _, _, err := ZoneName("en-US", zone, ms, "short"); err == nil {
				report("%s at %v: expected %v", zone, ms, f[1])
			}
		}
	}
	if bad > 0 {
		t.Errorf("%d zone name mismatches", bad)
	}
}
