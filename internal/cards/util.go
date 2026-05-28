package cards

import (
	"fmt"
	"strings"
)

func itoa(n int) string     { return fmt.Sprintf("%d", n) }
func ftoa(f float64) string { return fmt.Sprintf("%.2f", f) }
func cond(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}

// normalizeHex ensures a color string has a single leading #, defaulting grey
// for empty values. GitHub language colors usually already include the #.
func normalizeHex(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "#858585"
	}
	return "#" + strings.TrimPrefix(c, "#")
}
