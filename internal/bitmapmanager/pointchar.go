package bitmapmanager

// pointChar is a char at a point, for drawing a bitmap.
type pointChar struct {
	left int
	top  int
	char rune
}

// horizontalLine returns the points between beginExclusive and endExclusive, in that order.
func horizontalLine(beginExclusive, endExclusive, top int, char rune) []pointChar {
	var points []pointChar
	for _, left := range rangeExclusive(beginExclusive, endExclusive) {
		points = append(points, pointChar{left: left, top: top, char: char})
	}
	return points
}

// verticalLine returns the points between beginExclusive and endExclusive, in that order.
func verticalLine(left, beginExclusive, endExclusive int, char rune) []pointChar {
	var points []pointChar
	for _, top := range rangeExclusive(beginExclusive, endExclusive) {
		points = append(points, pointChar{left: left, top: top, char: char})
	}
	return points
}

func rangeExclusive(beginExclusive, endExclusive int) []int {
	if beginExclusive == endExclusive {
		return nil
	}
	delta := 1
	if beginExclusive > endExclusive {
		delta = -1
	}
	var values []int
	for value := beginExclusive + delta; value != endExclusive; value += delta {
		values = append(values, value)
	}
	return values
}
