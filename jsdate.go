package luxon

// The parts of the JavaScript Date object Luxon uses: Date.UTC, new Date(ts) with its UTC field
// getters, and setUTCFullYear, computed the way V8 computes them (time values are float64 epoch
// milliseconds, NaN outside ±8.64e15).

import "math"

const (
	msPerDay    = 86400000.0
	msPerHour   = 3600000.0
	msPerMinute = 60000.0
	msPerSecond = 1000.0
	maxTimeMs   = 8.64e15
)

// toIntegerOrInfinity is the abstract operation of that name (-0 becomes +0).
func toIntegerOrInfinity(x float64) float64 {
	if math.IsNaN(x) {
		return 0
	}
	if math.IsInf(x, 0) {
		return x
	}
	t := math.Trunc(x)
	if t == 0 {
		return 0
	}
	return t
}

var dayFromMonth = [2][12]int64{
	{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334},
	{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335},
}

// makeDay is V8's MakeDay over arguments already passed through ToNumber.
func makeDay(year, month, date float64) float64 {
	if math.IsNaN(year) || math.IsNaN(month) || math.IsNaN(date) || math.IsInf(date, 0) {
		return math.NaN()
	}
	year, month, date = toIntegerOrInfinity(year), toIntegerOrInfinity(month), toIntegerOrInfinity(date)
	if !(-1000000 <= year && year <= 1000000 && -10000000 <= month && month <= 10000000) {
		return math.NaN()
	}
	y, m := int64(year), int64(month)
	y += m / 12
	m %= 12
	if m < 0 {
		m += 12
		y--
	}
	const yearDelta = 399999
	base := 365*(1970+yearDelta) + (1970+yearDelta)/4 - (1970+yearDelta)/100 + (1970+yearDelta)/400
	day := 365*(y+yearDelta) + (y+yearDelta)/4 - (y+yearDelta)/100 + (y+yearDelta)/400 - int64(base)
	leap := 0
	if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
		leap = 1
	}
	day += dayFromMonth[leap][m]
	return float64(day-1) + date
}

// makeTime is MakeTime.
func makeTime(h, m, s, ms float64) float64 {
	for _, v := range []float64{h, m, s, ms} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return math.NaN()
		}
	}
	return float64(toIntegerOrInfinity(h)*msPerHour) + float64(toIntegerOrInfinity(m)*msPerMinute) +
		float64(toIntegerOrInfinity(s)*msPerSecond) + toIntegerOrInfinity(ms)
}

// makeDate is MakeDate.
func makeDate(day, time float64) float64 {
	if math.IsNaN(day) || math.IsInf(day, 0) || math.IsNaN(time) || math.IsInf(time, 0) {
		return math.NaN()
	}
	return float64(day*msPerDay) + time
}

// timeClip is TimeClip.
func timeClip(t float64) float64 {
	if math.IsNaN(t) || math.IsInf(t, 0) || math.Abs(t) > maxTimeMs {
		return math.NaN()
	}
	return toIntegerOrInfinity(t)
}

// dateUTC is Date.UTC(year, month, date, hours, minutes, seconds, ms).
func dateUTC(year, month, date, h, mi, s, ms float64) float64 {
	yr := year
	if !math.IsNaN(year) {
		yi := toIntegerOrInfinity(year)
		if yi >= 0 && yi <= 99 {
			yr = 1900 + yi
		}
	}
	return timeClip(makeDate(makeDay(yr, month, date), makeTime(h, mi, s, ms)))
}

// dateFields are the UTC getters of a Date whose time value is t (all NaN for an invalid date).
type dateFields struct {
	year, month0, date, hours, minutes, seconds, ms, weekday float64
}

func floorDivF(a, b float64) float64 { return math.Floor(a / b) }

func posMod(a, b float64) float64 {
	r := math.Mod(a, b)
	if r < 0 {
		r += b
	}
	return r
}

// utcFields reads the UTC fields of new Date(ts).
func utcFields(ts float64) dateFields {
	t := timeClip(ts)
	if math.IsNaN(t) {
		n := math.NaN()
		return dateFields{n, n, n, n, n, n, n, n}
	}
	day := floorDivF(t, msPerDay)
	within := t - day*msPerDay
	y, m, d := civilFromDays(int64(day))
	return dateFields{
		year:    float64(y),
		month0:  float64(m - 1),
		date:    float64(d),
		hours:   math.Floor(within / msPerHour),
		minutes: posMod(math.Floor(within/msPerMinute), 60),
		seconds: posMod(math.Floor(within/msPerSecond), 60),
		ms:      posMod(within, 1000),
		weekday: posMod(day+4, 7),
	}
}

// setUTCFullYear is new Date(t).setUTCFullYear(year, month, date) and returns the new time value.
func setUTCFullYear(t, year, month0, date float64) float64 {
	if math.IsNaN(t) {
		t = 0
	}
	day := floorDivF(t, msPerDay)
	within := t - day*msPerDay
	return timeClip(makeDate(makeDay(year, month0, date), within))
}

// civilFromDays converts days since 1970-01-01 to a proleptic Gregorian date.
func civilFromDays(z int64) (y, m, d int64) {
	z += 719468
	era := z / 146097
	if z < 0 && z%146097 != 0 {
		era--
	}
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
