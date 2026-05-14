package cmd

import (
	"reflect"
	"testing"
)

func TestParseWindowDays(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    []int
		wantErr bool
	}{
		{"defaults", "1,7,14,30,90", []int{1, 7, 14, 30, 90}, false},
		{"whitespace", " 1 , 7 , 14 ", []int{1, 7, 14}, false},
		{"dedup", "7,1,7,14,1", []int{1, 7, 14}, false},
		{"sorts", "30,1,7", []int{1, 7, 30}, false},
		{"single", "5", []int{5}, false},
		{"non-numeric", "1,abc,7", nil, true},
		{"zero", "0,1", nil, true},
		{"negative", "-1,7", nil, true},
		{"empty", "", nil, true},
		{"only-commas", ",,,", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseWindowDays(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil (got=%v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
