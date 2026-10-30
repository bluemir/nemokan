package bitmap

import "golang.org/x/text/width"

// WideCharTail is the second cell of a wide char. It hides what is below like a char, but
// is not drawn and is left out of strings. See ADR-0008.
const WideCharTail rune = 2

// CharWidth returns 2 for East Asian Wide and Fullwidth chars, such as Hangul, otherwise 1.
// Ambiguous chars are 1. See ADR-0008.
func CharWidth(char rune) int {
	switch width.LookupRune(char).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

// StringWidth returns the number of cells text takes.
func StringWidth(text string) int {
	total := 0
	for _, char := range text {
		total += CharWidth(char)
	}
	return total
}
