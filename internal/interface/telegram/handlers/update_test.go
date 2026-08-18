package handlers

import (
	"testing"

	"github.com/blang/semver"
)

func TestParseVersionFourPart(t *testing.T) {
	cases := []struct {
		in   string
		want string // semver.String()
	}{
		{"v26.5.2.5", "26.5.2-5"},
		{"26.5.2.0", "26.5.2-0"},
		{"26.5.3.0", "26.5.3-0"},
		{"v26.4.3.0", "26.4.3-0"},
		{"26.5.2", "26.5.2"},
	}
	for _, c := range cases {
		v, err := parseVersion(c.in)
		if err != nil {
			t.Fatalf("parseVersion(%q) err: %v", c.in, err)
		}
		if v.String() != c.want {
			t.Fatalf("parseVersion(%q) = %s, want %s", c.in, v.String(), c.want)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	must := func(s string) semver.Version {
		v, err := parseVersion(s)
		if err != nil {
			t.Fatalf("parse %s: %v", s, err)
		}
		return v
	}
	if !must("26.5.2.5").GT(must("26.5.2.0")) {
		t.Error("26.5.2.5 should be > 26.5.2.0")
	}
	if !must("26.5.3.0").GT(must("26.5.2.5")) {
		t.Error("26.5.3.0 should be > 26.5.2.5")
	}
	if !must("26.5.2.6").GT(must("26.5.2.5")) {
		t.Error("26.5.2.6 should be > 26.5.2.5")
	}
	if must("26.5.2.5").GT(must("26.5.2.5")) {
		t.Error("26.5.2.5 should not be > itself")
	}
	if !must("26.5.2.5").EQ(must("26.5.2.5")) {
		t.Error("26.5.2.5 should equal itself")
	}
}
