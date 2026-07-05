//go:build windows

package brightness

import "testing"

// The percent→raw mapping is the piece DDC/CI silently gets wrong on monitors
// whose VCP range is not 0-100; pin it down.
func TestDDCRaw(t *testing.T) {
	cases := []struct {
		level    int
		min, max uint32
		want     uint32
	}{
		{0, 0, 100, 0},
		{50, 0, 100, 50},
		{100, 0, 100, 100},
		{50, 0, 255, 127},
		{100, 0, 255, 255},
		{50, 20, 120, 70},
		{0, 20, 120, 20},
		{75, 0, 0, 75}, // degenerate range falls back to raw percent
	}
	for _, c := range cases {
		if got := ddcRaw(c.level, c.min, c.max); got != c.want {
			t.Errorf("ddcRaw(%d, %d, %d) = %d, want %d", c.level, c.min, c.max, got, c.want)
		}
	}
}
