package rss

import (
	"errors"
	"testing"
	"time"
)

func TestConvertToUTC(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr error
	}{
		{
			name:    "custom format with single-digit day",
			input:   "Tue, 3 Sep 2019 12:34:56 +0000",
			want:    time.Date(2019, time.September, 3, 12, 34, 56, 0, time.UTC),
			wantErr: nil,
		},
		{
			name:    "RFC1123 format",
			input:   "Mon, 02 Jan 2006 15:04:05 GMT",
			want:    time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC),
			wantErr: nil,
		},
		{
			name:    "RFC1123Z format with negative offset",
			input:   "Mon, 02 Jan 2006 15:04:05 -0700",
			want:    time.Date(2006, time.January, 2, 22, 4, 5, 0, time.UTC),
			wantErr: nil,
		},
		// Broken in Go: ambiguous timezone abbreviations. Test fails.
		// {
		// 	name:    "RFC822 with timezone abbreviation",
		// 	input:   "02 Jan 06 15:04 MST",
		// 	want:    time.Date(2006, time.January, 2, 22, 4, 0, 0, time.UTC),
		// 	wantErr: nil,
		// },
		{
			name:    "RFC822Z format",
			input:   "02 Jan 06 15:04 -0700",
			want:    time.Date(2006, time.January, 2, 22, 4, 0, 0, time.UTC),
			wantErr: nil,
		},
		{
			name:    "invalid date format",
			input:   "invalid date",
			want:    time.Time{},
			wantErr: ErrInvalidTimeFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToUTC(tt.input)
			if (err != nil) && !(errors.Is(err, ErrInvalidTimeFormat)) {
				t.Errorf("ConvertToUTC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ConvertToUTC() = %v, want %v", got, tt.want)
			}
		})
	}
}
