package web

import "testing"

func TestSafeRedirectTarget(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/games/new", "/games/new"},
		{"//evil.com", "/"},
		{"https://evil.com", "/"},
		{"http://evil.com/x", "/"},
		{"/", "/"},
	}

	for _, c := range cases {
		if got := safeRedirectTarget(c.in); got != c.want {
			t.Errorf("safeRedirectTarget(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
