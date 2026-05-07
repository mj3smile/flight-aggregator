package domain

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		minutes int
		want    string
	}{
		{0, "0m"},
		{45, "45m"},
		{60, "1h"},
		{110, "1h 50m"},
		{260, "4h 20m"},
		{120, "2h"},
	}
	for _, tt := range tests {
		if got := FormatDuration(tt.minutes); got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.minutes, got, tt.want)
		}
	}
}

func TestFormatIDR(t *testing.T) {
	tests := []struct {
		amount int
		want   string
	}{
		{500, "IDR 500"},
		{1000, "IDR 1,000"},
		{485000, "IDR 485,000"},
		{1250000, "IDR 1,250,000"},
		{95000000, "IDR 95,000,000"},
	}
	for _, tt := range tests {
		if got := FormatIDR(tt.amount); got != tt.want {
			t.Errorf("FormatIDR(%d) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

func TestCityForAirport(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"CGK", "Jakarta"},
		{"DPS", "Denpasar"},
		{"SUB", "Surabaya"},
		{"XYZ", "XYZ"},
	}
	for _, tt := range tests {
		if got := CityForAirport(tt.code); got != tt.want {
			t.Errorf("CityForAirport(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
