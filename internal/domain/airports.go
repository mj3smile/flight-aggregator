package domain

var airportCity = map[string]string{
	"CGK":  "Jakarta",
	"DPS":  "Denpasar",
	"SUB":  "Surabaya",
	"UPG":  "Makassar",
	"SOC":  "Solo",
	"JOG":  "Yogyakarta",
	"BDO":  "Bandung",
	"MDC":  "Manado",
	"BPN":  "Balikpapan",
	"PLM":  "Palembang",
	"KNO":  "Medan",
	"PDG":  "Padang",
	"PKU":  "Pekanbaru",
	"BTH":  "Batam",
	"SRG":  "Semarang",
	"LOP":  "Lombok",
	"AMQ":  "Ambon",
	"KDI":  "Kendari",
	"TKG":  "Bandar Lampung",
	"HLP":  "Jakarta",
	"PNK":  "Pontianak",
}

func ValidAirport(code string) bool {
	_, ok := airportCity[code]
	return ok
}

func CityForAirport(code string) string {
	if city, ok := airportCity[code]; ok {
		return city
	}
	return code
}
