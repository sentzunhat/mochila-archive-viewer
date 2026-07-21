package archive

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.2.0", "0.1.0", true},
		{"0.1.0", "0.2.0", false},
		{"1.0.0", "1.0.0", false},
		{"0.10.0", "0.9.0", true}, // numeric, not lexicographic — this is why atoiSafe exists
		{"0.1", "0.1.0", false},   // short tag, treated as 0.1.0
	}
	for _, tc := range cases {
		if got := isNewer(tc.a, tc.b); got != tc.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCheckForUpdateDevBuild(t *testing.T) {
	// "dev" builds must never report an update, and must never make a
	// network call (verified implicitly — this test has no network access
	// assumptions and still must pass).
	status := CheckForUpdate("dev")
	if status.Available {
		t.Errorf("CheckForUpdate(\"dev\") reported available=true, want false")
	}
}
