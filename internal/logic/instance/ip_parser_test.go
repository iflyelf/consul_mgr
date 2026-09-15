package instance

import (
	"testing"
)

func TestParseIPPorts(t *testing.T) {
	cases := []struct {
		in   string
		want int
		head string
	}{
		{"10.1.255.24-26,10.1.255.38,10.1.255.0/24,10.1.26.5-10.1.26.7", 3 + 1 + 254 + 3, "10.1.255.24"},
		{"10.1.255.24-26:80,10.1.255.38:443", 4, "10.1.255.24:80"},
		{"36.7.68.246:443,36.7.68.246:888", 2, "36.7.68.246:443"},
		{"1.1.1.1:80", 1, "1.1.1.1:80"},
		{"10.1.255.0/30:8080", 2, "10.1.255.1:8080"},
	}
	for _, c := range cases {
		got, err := ParseIPPorts(c.in)
		if err != nil {
			t.Fatalf("ParseIPPorts(%q) error: %v", c.in, err)
		}
		if len(got) != c.want {
			t.Errorf("ParseIPPorts(%q) len=%d want=%d (got %v)", c.in, len(got), c.want, got)
		}
		if len(got) > 0 && got[0] != c.head {
			t.Errorf("ParseIPPorts(%q) head=%q want=%q", c.in, got[0], c.head)
		}
	}
}
