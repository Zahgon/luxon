package intl

import (
	"math"
	"sort"
	"strconv"
	"sync"
)

// calFields are the calendar fields ICU formats for one local date. Era and Month are the keys
// used by the pattern tables: the English short era name and the English month name (Hebrew months
// carry a "|0" or "|1" leap-year suffix).
type calFields struct {
	Era   string
	Year  int64
	Month string
	Day   int64
}

type calData struct {
	JapaneseEras  [][2]any            `json:"japaneseEras"`
	Umalqura      [][3]any            `json:"umalqura"`
	MonthNames    map[string][]string `json:"monthNames"`
	PersianStarts struct {
		First  int64   `json:"first"`
		Starts []int64 `json:"starts"`
	} `json:"persianStarts"`
}

type japaneseEra struct {
	start     int64
	name      string
	startYear int64
}

type uqMonth struct {
	start int64
	year  int64
	month string
}

var (
	calOnce  sync.Once
	cal      calData
	japEras  []japaneseEra
	uqMonths []uqMonth
)

func caldata() {
	calOnce.Do(func() {
		loadJSON("cal", &cal)
		for i, e := range cal.JapaneseEras {
			d := int64(e[0].(float64))
			era := japaneseEra{start: d, name: e[1].(string)}
			if i == 0 {
				era.startYear = 645
			} else {
				era.startYear, _, _ = gregorianJulianFields(d + epochJD)
			}
			japEras = append(japEras, era)
		}
		for _, m := range cal.Umalqura {
			y, _ := strconv.ParseInt(m[1].(string), 10, 64)
			uqMonths = append(uqMonths, uqMonth{start: int64(m[0].(float64)), year: y, month: m[2].(string)})
		}
	})
}

const (
	epochJD        = 2440588 // ICU Julian day of 1970-01-01
	gregorianShift = 2299161 // ICU Julian day of 1582-10-15
)

var (
	monthsGregorian = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	monthsPersian   = []string{"Farvardin", "Ordibehesht", "Khordad", "Tir", "Mordad", "Shahrivar", "Mehr", "Aban", "Azar", "Dey", "Bahman", "Esfand"}
	monthsIslamic   = []string{"Muharram", "Safar", "Rabiʻ I", "Rabiʻ II", "Jumada I", "Jumada II", "Rajab", "Shaʻban", "Ramadan", "Shawwal", "Dhuʻl-Qiʻdah", "Dhuʻl-Hijjah"}
	monthsIndian    = []string{"Chaitra", "Vaisakha", "Jyaistha", "Asadha", "Sravana", "Bhadra", "Asvina", "Kartika", "Agrahayana", "Pausa", "Magha", "Phalguna"}
	monthsCoptic    = []string{"Tout", "Baba", "Hator", "Kiahk", "Toba", "Amshir", "Baramhat", "Baramouda", "Bashans", "Paona", "Epep", "Mesra", "Nasie"}
	monthsEthiopic  = []string{"Meskerem", "Tekemt", "Hedar", "Tahsas", "Ter", "Yekatit", "Megabit", "Miazia", "Genbot", "Sene", "Hamle", "Nehasse", "Pagumen"}
	monthsHebrew    = []string{"Tishri", "Heshvan", "Kislev", "Tevet", "Shevat", "Adar I", "Adar", "Nisan", "Iyar", "Sivan", "Tamuz", "Av", "Elul"}
)

var cumDays = [2][12]int64{
	{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334},
	{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335},
}

// gregorianJulianFields is ICU's GregorianCalendar field computation with the default
// 1582-10-15 switch from the Julian calendar.
func gregorianJulianFields(jd int64) (eyear, month, day int64) {
	if jd >= gregorianShift {
		y, m, d := civilFromDays(jd - epochJD)
		return y, m, d
	}
	jed := jd - (1721426 - 2)
	eyear = int64(math.Floor((4*float64(jed) + 1464) / 1461))
	jan1 := 365*(eyear-1) + floorDiv(eyear-1, 4)
	doy := jed - jan1
	leap := eyear&3 == 0
	corr := int64(0)
	march1 := int64(59)
	if leap {
		march1 = 60
	}
	if doy >= march1 {
		if leap {
			corr = 1
		} else {
			corr = 2
		}
	}
	m := (12*(doy+corr) + 6) / 367
	li := 0
	if leap {
		li = 1
	}
	return eyear, m + 1, doy - cumDays[li][m] + 1
}

func calendarFields(calendar string, day int64) calFields {
	jd := day + epochJD
	switch calendar {
	case "gregory", "iso8601":
		y, m, d := civilFromDays(day)
		return gregorianEra(y, m, d)
	case "buddhist":
		y, m, d := gregorianJulianFields(jd)
		return calFields{Era: "BE", Year: y + 543, Month: monthsGregorian[m-1], Day: d}
	case "roc":
		y, m, d := gregorianJulianFields(jd)
		y -= 1911
		if y > 0 {
			return calFields{Era: "Minguo", Year: y, Month: monthsGregorian[m-1], Day: d}
		}
		return calFields{Era: "B.R.O.C.", Year: 1 - y, Month: monthsGregorian[m-1], Day: d}
	case "japanese":
		caldata()
		y, m, d := gregorianJulianFields(jd)
		i := sort.Search(len(japEras), func(i int) bool { return japEras[i].start > day }) - 1
		if i < 0 {
			i = 0
		}
		e := japEras[i]
		return calFields{Era: e.name, Year: y - e.startYear + 1, Month: monthsGregorian[m-1], Day: d}
	case "persian":
		return persianFields(jd)
	case "hebrew":
		return hebrewFields(jd)
	case "indian":
		return indianFields(day)
	case "coptic":
		y, m, d := ceFields(jd, 1824665)
		if y <= 0 {
			return calFields{Era: "", Year: 1 - y, Month: monthsCoptic[m], Day: d}
		}
		return calFields{Era: "AM", Year: y, Month: monthsCoptic[m], Day: d}
	case "ethiopic":
		y, m, d := ceFields(jd, 1723856)
		if y > 0 {
			return calFields{Era: "AM", Year: y, Month: monthsEthiopic[m], Day: d}
		}
		return calFields{Era: "AA", Year: y + 5500, Month: monthsEthiopic[m], Day: d}
	case "ethioaa":
		y, m, d := ceFields(jd, 1723856)
		return calFields{Era: "AA", Year: y + 5500, Month: monthsEthiopic[m], Day: d}
	case "islamic-civil":
		return islamicCivil(jd, 1948440)
	case "islamic-tbla":
		return islamicCivil(jd, 1948439)
	case "islamic-umalqura":
		caldata()
		if len(uqMonths) > 0 && day >= uqMonths[0].start && day < uqMonths[len(uqMonths)-1].start {
			i := sort.Search(len(uqMonths), func(i int) bool { return uqMonths[i].start > day }) - 1
			u := uqMonths[i]
			return calFields{Era: "AH", Year: u.year, Month: u.month, Day: day - u.start + 1}
		}
		return islamicCivil(jd, 1948440)
	}
	y, m, d := civilFromDays(day)
	return gregorianEra(y, m, d)
}

func gregorianEra(y, m, d int64) calFields {
	if y <= 0 {
		return calFields{Era: "BC", Year: 1 - y, Month: monthsGregorian[m-1], Day: d}
	}
	return calFields{Era: "AD", Year: y, Month: monthsGregorian[m-1], Day: d}
}

var persianCum = [12]int64{0, 31, 62, 93, 124, 155, 186, 216, 246, 276, 306, 336}

func persianFields(jd int64) calFields {
	caldata()
	since := jd - 1948320
	var year, doy int64
	if t := cal.PersianStarts; len(t.Starts) > 1 && jd-epochJD >= t.Starts[0] && jd-epochJD < t.Starts[len(t.Starts)-1] {
		day := jd - epochJD
		i := sort.Search(len(t.Starts), func(i int) bool { return t.Starts[i] > day }) - 1
		year, doy = t.First+int64(i), day-t.Starts[i]
	} else {
		year = 1 + floorDiv(33*since+3, 12053)
		doy = since - (365*(year-1) + floorDiv(8*year+21, 33))
	}
	var month int64
	if doy < 216 {
		month = doy / 31
	} else {
		month = (doy - 6) / 30
	}
	return calFields{Era: "AP", Year: year, Month: monthsPersian[month], Day: doy - persianCum[month] + 1}
}

func ceFields(jd, epoch int64) (year, month, day int64) {
	c4 := floorDiv(jd-epoch, 1461)
	r4 := floorMod(jd-epoch, 1461)
	year = 4*c4 + (r4/365 - r4/1460)
	doy := r4 % 365
	if r4 == 1460 {
		doy = 365
	}
	return year, doy / 30, doy%30 + 1
}

func islamicYearStart(year int64) int64 { return (year-1)*354 + floorDiv(3+11*year, 30) }

func islamicCivil(jd, epoch int64) calFields {
	days := jd - epoch
	year := floorDiv(30*days+10646, 10631)
	month := int64(math.Ceil(float64(days-29-islamicYearStart(year)) / 29.5))
	if month > 11 {
		month = 11
	}
	start := int64(math.Ceil(29.5*float64(month))) + islamicYearStart(year)
	return calFields{Era: "AH", Year: year, Month: monthsIslamic[month], Day: days - start + 1}
}

func indianFields(day int64) calFields {
	gy, _, _ := civilFromDays(day)
	year := gy - 78
	yday := day - daysFromCivil(gy, 1, 1)
	leap := func(y int64) bool { return (y%4 == 0 && y%100 != 0) || y%400 == 0 }
	var leapMonth int64
	if yday < 80 {
		year--
		leapMonth = 30
		if leap(gy - 1) {
			leapMonth = 31
		}
		yday += leapMonth + 31*5 + 30*3 + 10
	} else {
		leapMonth = 30
		if leap(gy) {
			leapMonth = 31
		}
		yday -= 80
	}
	var month, dom int64
	if yday < leapMonth {
		month, dom = 0, yday+1
	} else {
		mday := yday - leapMonth
		if mday < 31*5 {
			month, dom = mday/31+1, mday%31+1
		} else {
			mday -= 31 * 5
			month, dom = mday/30+6, mday%30+1
		}
	}
	return calFields{Era: "Śaka", Year: year, Month: monthsIndian[month], Day: dom}
}

const (
	hHourParts  = 1080
	hDayParts   = 24 * hHourParts
	hMonthFract = 12*hHourParts + 793
	hMonthParts = 29*hDayParts + hMonthFract
	hBaharad    = 11*hHourParts + 204
)

var hebrewMonthStart = [14][3]int64{
	{0, 0, 0}, {30, 30, 30}, {59, 59, 60}, {88, 89, 90}, {117, 118, 119}, {147, 148, 149},
	{147, 148, 149}, {176, 177, 178}, {206, 207, 208}, {235, 236, 237}, {265, 266, 267},
	{294, 295, 296}, {324, 325, 326}, {353, 354, 355},
}

var hebrewLeapMonthStart = [14][3]int64{
	{0, 0, 0}, {30, 30, 30}, {59, 59, 60}, {88, 89, 90}, {117, 118, 119}, {147, 148, 149},
	{177, 178, 179}, {206, 207, 208}, {236, 237, 238}, {265, 266, 267}, {295, 296, 297},
	{324, 325, 326}, {354, 355, 356}, {383, 384, 385},
}

func hebrewLeap(year int64) bool {
	x := (year*12 + 17) % 19
	if x < 0 {
		return x >= -7
	}
	return x >= 12
}

var (
	hebrewCacheMu sync.Mutex
	hebrewCache   = map[int64]int64{}
)

func hebrewStartOfYear(year int64) int64 {
	hebrewCacheMu.Lock()
	if d, ok := hebrewCache[year]; ok {
		hebrewCacheMu.Unlock()
		return d
	}
	hebrewCacheMu.Unlock()
	months := floorDiv(235*year-234, 19)
	frac := months*hMonthFract + hBaharad
	day := months*29 + frac/hDayParts
	frac = frac % hDayParts
	switch wd := day % 7; {
	case wd == 2 || wd == 4 || wd == 6:
		day++
	case wd == 1 && frac > 15*hHourParts+204 && !hebrewLeap(year):
		day += 2
	case wd == 0 && frac > 21*hHourParts+589 && hebrewLeap(year-1):
		day++
	}
	hebrewCacheMu.Lock()
	hebrewCache[year] = day
	hebrewCacheMu.Unlock()
	return day
}

func hebrewYearType(year int64) int {
	l := hebrewStartOfYear(year+1) - hebrewStartOfYear(year)
	if l > 380 {
		l -= 30
	}
	switch l {
	case 353:
		return 0
	case 355:
		return 2
	}
	return 1
}

func hebrewFields(jd int64) calFields {
	d := jd - 347997
	m := math.Floor(float64(d) * hDayParts / hMonthParts)
	year := int64(math.Floor((float64(19*m)+234)/235) + 1)
	ys := hebrewStartOfYear(year)
	doy := d - ys
	for doy < 1 {
		year--
		ys = hebrewStartOfYear(year)
		doy = d - ys
	}
	typ := hebrewYearType(year)
	leap := hebrewLeap(year)
	table := &hebrewMonthStart
	if leap {
		table = &hebrewLeapMonthStart
	}
	month := 0
	for month < 14 && doy > table[month][typ] {
		month++
	}
	month--
	if month < 0 {
		month = 0
	}
	dom := doy - table[month][typ]
	name := monthsHebrew[month]
	flag := "|0"
	if leap {
		flag = "|1"
		if month == 6 {
			name = "Adar II"
		}
	}
	return calFields{Era: "AM", Year: year, Month: name + flag, Day: dom}
}

// isoWeekday returns 1 (Monday) .. 7 (Sunday) for days since 1970-01-01.
func isoWeekday(day int64) int64 { return floorMod(day+3, 7) + 1 }
