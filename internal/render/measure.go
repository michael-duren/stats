package render

// MeasureText approximates the rendered pixel width of a string for a given
// font size. The actual font is the viewer's system font (Segoe UI / Verdana),
// so exact metrics are impossible server-side; this average-advance estimate is
// good enough for centering and auto-width decisions, matching the approach
// github-readme-stats uses.
func MeasureText(s string, fontSize float64) float64 {
	var width float64
	for _, r := range s {
		width += charWidth(r)
	}
	return width * fontSize
}

// charWidth returns the advance of a rune as a fraction of the font size,
// tuned for a typical proportional UI font.
func charWidth(r rune) float64 {
	switch {
	case r == ' ':
		return 0.28
	case r == 'i' || r == 'l' || r == 'j' || r == '.' || r == ',' || r == '\'' || r == '|' || r == '!':
		return 0.27
	case r == 'm' || r == 'w' || r == 'M' || r == 'W':
		return 0.86
	case r >= 'A' && r <= 'Z':
		return 0.66
	case r >= '0' && r <= '9':
		return 0.56
	default:
		return 0.53
	}
}
