package cli

import (
	"testing"
)

func TestParseFailRate(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"20", 20.0, false},
		{"20%", 20.0, false},
		{"100", 100.0, false},
		{"-5", 0, true},
		{"150", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseFailRate(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFailRate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseFailRate(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseStatusFlags(t *testing.T) {
	tests := []struct {
		input   []string
		wantLen int
		wantErr bool
	}{
		{[]string{"429:20", "503:10%"}, 2, false},
		{[]string{"500:5"}, 1, false},
		{[]string{"invalid"}, 0, true},
		{[]string{"500:abc"}, 0, true},
		{[]string{"999:10"}, 0, true}, // Invalid HTTP code
	}

	for _, tt := range tests {
		got, err := ParseStatusFlags(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseStatusFlags(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && len(got) != tt.wantLen {
			t.Errorf("ParseStatusFlags(%v) len = %v, want %v", tt.input, len(got), tt.wantLen)
		}
	}
}

func TestParseLatency(t *testing.T) {
	tests := []struct {
		input   string
		wantDist string
		wantErr bool
	}{
		{"2s", "", false},
		{"normal:500ms,2s", "normal", false},
		{"uniform:1s,3s", "uniform", false},
		{"invalid:1s,3s", "", true},
		{"normal:1s", "", true}, // missing max
	}

	for _, tt := range tests {
		got, err := ParseLatency(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseLatency(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got.Distribution != tt.wantDist {
			t.Errorf("ParseLatency(%q) dist = %v, want %v", tt.input, got.Distribution, tt.wantDist)
		}
		if !tt.wantErr && tt.wantDist == "" && got.Fixed == 0 {
			t.Errorf("ParseLatency(%q) fixed latency should not be 0", tt.input)
		}
	}
}
