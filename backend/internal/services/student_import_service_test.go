package services

import (
	"testing"
)

func TestNormalizeGender(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"M", "M"}, {"male", "M"}, {"BOY", "M"},
		{"F", "F"}, {"female", "F"}, {"Girl", "F"},
		{"", "OTHER"}, {"unknown", "OTHER"},
	}
	for _, c := range cases {
		if got := NormalizeGender(c.in); got != c.want {
			t.Errorf("NormalizeGender(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		in   string
		want string // formatted YYYY-MM-DD
		ok   bool
	}{
		{"2006-01-02", "2006-01-02", true},
		{"02/01/2006", "2006-01-02", true},
		{"01-02-2006", "2006-01-02", true},
		{"invalid", "", false},
	}
	for _, c := range cases {
		got, err := ParseDate(c.in)
		if c.ok {
			if err != nil {
				t.Errorf("ParseDate(%q) error %v", c.in, err)
				continue
			}
			if got.Format("2006-01-02") != c.want {
				t.Errorf("ParseDate(%q) = %v, want %s", c.in, got, c.want)
			}
		} else {
			if err == nil {
				t.Errorf("ParseDate(%q) expected error", c.in)
			}
		}
	}
}

func TestParseDate_RFC3339(t *testing.T) {
	_, err := ParseDate("2006-01-02T15:04:05Z")
	if err != nil {
		t.Errorf("ParseDate RFC3339 failed: %v", err)
	}
}

func TestParseDate_Empty(t *testing.T) {
	_, err := ParseDate("")
	if err == nil {
		t.Errorf("ParseDate empty expected error")
	}
}
