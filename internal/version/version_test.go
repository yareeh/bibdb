package version

import "testing"

func TestGTE(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.4.0", "1.3.1", true},
		{"1.3.1", "1.4.0", false},
		{"1.4.0", "1.4.0", true},
		{"v1.4.0", "1.4.0", true},
		{"2.0.0", "1.99.99", true},
		{"1.4.0", "1.4", true},
		{"1.4", "1.4.0", true},
		{"1.4.0-rc1", "1.4.0", true}, // pre-release suffix ignored
		{"", "0.0.0", true},
		{"0.0.0", "", true},
		{"1.4.0", "", true},
		{"", "1.4.0", false},
	}
	for _, c := range cases {
		if got := GTE(c.a, c.b); got != c.want {
			t.Errorf("GTE(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCurrentFallsBackToConstant(t *testing.T) {
	// In `go test` the build info reports a version like "(devel)" or empty.
	// Current() must therefore return Constant.
	got := Current()
	if got != Constant {
		t.Errorf("Current() = %q, want %q (Constant) — runtime build info should not be set during go test", got, Constant)
	}
}
