package models

import (
	"testing"
	"time"
)

func TestPeriodCutoff(t *testing.T) {
	now := time.Date(2026, 2, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		period  string
		want    time.Time
		wantErr bool
	}{
		{"day", time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC), false},
		{"week", time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC), false},
		{"month", time.Date(2026, 1, 18, 12, 0, 0, 0, time.UTC), false},
		{"", time.Time{}, true},
		{"7d", time.Time{}, true},
		{"year", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			got, err := PeriodCutoff(tt.period, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("PeriodCutoff(%q) error = %v, wantErr %v", tt.period, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("PeriodCutoff(%q) = %v, want %v", tt.period, got, tt.want)
			}
		})
	}
}
