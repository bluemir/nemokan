package board

import "math/bits"

// Each direction mask has 4 bits (left, right, top, bottom) per line style: single, bold, double.
const (
	maskSingleLeft       = 0b0001
	maskSingleRight      = 0b0010
	maskSingleTop        = 0b0100
	maskSingleBottom     = 0b1000
	maskSingleHorizontal = maskSingleLeft | maskSingleRight
	maskSingleVertical   = maskSingleTop | maskSingleBottom
	maskSingleCross      = maskSingleHorizontal | maskSingleVertical

	maskBoldLeft       = maskSingleLeft << 4
	maskBoldRight      = maskSingleRight << 4
	maskBoldTop        = maskSingleTop << 4
	maskBoldBottom     = maskSingleBottom << 4
	maskBoldHorizontal = maskSingleHorizontal << 4
	maskBoldVertical   = maskSingleVertical << 4
	maskBoldCross      = maskSingleCross << 4

	maskDoubleLeft       = maskSingleLeft << 8
	maskDoubleRight      = maskSingleRight << 8
	maskDoubleTop        = maskSingleTop << 8
	maskDoubleBottom     = maskSingleBottom << 8
	maskDoubleHorizontal = maskSingleHorizontal << 8
	maskDoubleVertical   = maskSingleVertical << 8
	maskDoubleCross      = maskSingleCross << 8

	maskLeft   = maskSingleLeft | maskBoldLeft | maskDoubleLeft
	maskRight  = maskSingleRight | maskBoldRight | maskDoubleRight
	maskTop    = maskSingleTop | maskBoldTop | maskDoubleTop
	maskBottom = maskSingleBottom | maskBoldBottom | maskDoubleBottom
	maskCross  = maskSingleCross | maskBoldCross | maskDoubleCross
)

var standardizedChars = map[rune]rune{
	'-': '─',
	'|': '│',
	'+': '┼',
	'╮': '┐',
	'╭': '┌',
	'╯': '┘',
	'╰': '└',
}

var charToMask = map[rune]int{
	'─': maskSingleHorizontal,
	'│': maskSingleVertical,
	'┘': maskSingleLeft | maskSingleTop,
	'┐': maskSingleLeft | maskSingleBottom,
	'┤': maskSingleLeft | maskSingleVertical,
	'└': maskSingleRight | maskSingleTop,
	'┌': maskSingleRight | maskSingleBottom,
	'├': maskSingleRight | maskSingleVertical,
	'┴': maskSingleHorizontal | maskSingleTop,
	'┬': maskSingleHorizontal | maskSingleBottom,
	'┼': maskSingleHorizontal | maskSingleVertical,

	'━': maskBoldHorizontal,
	'┃': maskBoldVertical,
	'┛': maskBoldLeft | maskBoldTop,
	'┓': maskBoldLeft | maskBoldBottom,
	'┫': maskBoldLeft | maskBoldVertical,
	'┗': maskBoldRight | maskBoldTop,
	'┏': maskBoldRight | maskBoldBottom,
	'┣': maskBoldRight | maskBoldVertical,
	'┻': maskBoldHorizontal | maskBoldTop,
	'┳': maskBoldHorizontal | maskBoldBottom,
	'╋': maskBoldHorizontal | maskBoldVertical,

	'═': maskDoubleHorizontal,
	'║': maskDoubleVertical,
	'╝': maskDoubleLeft | maskDoubleTop,
	'╗': maskDoubleLeft | maskDoubleBottom,
	'╣': maskDoubleLeft | maskDoubleVertical,
	'╚': maskDoubleRight | maskDoubleTop,
	'╔': maskDoubleRight | maskDoubleBottom,
	'╠': maskDoubleRight | maskDoubleVertical,
	'╩': maskDoubleHorizontal | maskDoubleTop,
	'╦': maskDoubleHorizontal | maskDoubleBottom,
	'╬': maskDoubleHorizontal | maskDoubleVertical,

	// Complex (single, bold) combinations
	'╼': maskSingleLeft | maskBoldRight,
	'╾': maskBoldLeft | maskSingleRight,

	'╽': maskSingleTop | maskBoldBottom,
	'╿': maskBoldTop | maskSingleBottom,

	'┚': maskSingleLeft | maskBoldTop,
	'┙': maskBoldLeft | maskSingleTop,

	'┒': maskSingleLeft | maskBoldBottom,
	'┑': maskBoldLeft | maskSingleBottom,

	'┨': maskSingleLeft | maskBoldTop | maskBoldBottom,
	'┦': maskSingleLeft | maskBoldTop | maskSingleBottom,
	'┧': maskSingleLeft | maskSingleTop | maskBoldBottom,

	'┥': maskBoldLeft | maskSingleTop | maskSingleBottom,
	'┩': maskBoldLeft | maskBoldTop | maskSingleBottom,
	'┪': maskBoldLeft | maskSingleTop | maskBoldBottom,

	'┖': maskSingleRight | maskBoldTop,
	'┕': maskBoldRight | maskSingleTop,

	'┎': maskSingleRight | maskBoldBottom,
	'┍': maskBoldRight | maskSingleBottom,

	'┠': maskSingleRight | maskBoldTop | maskBoldBottom,
	'┞': maskSingleRight | maskBoldTop | maskSingleBottom,
	'┟': maskSingleRight | maskSingleTop | maskBoldBottom,

	'┝': maskBoldRight | maskSingleTop | maskSingleBottom,
	'┡': maskBoldRight | maskBoldTop | maskSingleBottom,
	'┢': maskBoldRight | maskSingleTop | maskBoldBottom,

	'┷': maskBoldLeft | maskBoldRight | maskSingleTop,
	'┶': maskSingleLeft | maskBoldRight | maskSingleTop,
	'┵': maskBoldLeft | maskSingleRight | maskSingleTop,

	'┸': maskSingleLeft | maskSingleRight | maskBoldTop,
	'┹': maskBoldLeft | maskSingleRight | maskBoldTop,
	'┺': maskSingleLeft | maskBoldRight | maskBoldTop,

	'┯': maskBoldLeft | maskBoldRight | maskSingleBottom,
	'┭': maskBoldLeft | maskSingleRight | maskSingleBottom,
	'┮': maskSingleLeft | maskBoldRight | maskSingleBottom,

	'┰': maskSingleLeft | maskSingleRight | maskBoldBottom,
	'┱': maskBoldLeft | maskSingleRight | maskBoldBottom,
	'┲': maskSingleLeft | maskBoldRight | maskBoldBottom,

	'┽': maskBoldLeft | maskSingleRight | maskSingleTop | maskSingleBottom,
	'┾': maskSingleLeft | maskBoldRight | maskSingleTop | maskSingleBottom,
	'╀': maskSingleLeft | maskSingleRight | maskBoldTop | maskSingleBottom,
	'╁': maskSingleLeft | maskSingleRight | maskSingleTop | maskBoldBottom,

	'╂': maskSingleLeft | maskSingleRight | maskBoldTop | maskBoldBottom,
	'┿': maskBoldLeft | maskBoldRight | maskSingleTop | maskSingleBottom,
	'╃': maskBoldLeft | maskSingleRight | maskBoldTop | maskSingleBottom,
	'╄': maskSingleLeft | maskBoldRight | maskBoldTop | maskSingleBottom,
	'╅': maskBoldLeft | maskSingleRight | maskSingleTop | maskBoldBottom,
	'╆': maskSingleLeft | maskBoldRight | maskSingleTop | maskBoldBottom,

	'╇': maskBoldLeft | maskBoldRight | maskBoldTop | maskSingleBottom,
	'╈': maskBoldLeft | maskBoldRight | maskSingleTop | maskBoldBottom,
	'╉': maskBoldLeft | maskSingleRight | maskBoldTop | maskBoldBottom,
	'╊': maskSingleLeft | maskBoldRight | maskBoldTop | maskBoldBottom,

	// Complex (single, double) combinations
	'╒': maskDoubleRight | maskSingleBottom,
	'╓': maskSingleRight | maskDoubleBottom,

	'╕': maskDoubleLeft | maskSingleBottom,
	'╖': maskSingleLeft | maskDoubleBottom,

	'╘': maskDoubleRight | maskSingleTop,
	'╙': maskSingleRight | maskDoubleTop,

	'╛': maskDoubleLeft | maskSingleTop,
	'╜': maskSingleLeft | maskDoubleTop,

	'╞': maskDoubleRight | maskSingleTop | maskSingleBottom,
	'╟': maskSingleRight | maskDoubleTop | maskDoubleBottom,

	'╡': maskDoubleLeft | maskSingleTop | maskSingleBottom,
	'╢': maskSingleLeft | maskDoubleTop | maskDoubleBottom,

	'╤': maskDoubleLeft | maskDoubleRight | maskSingleBottom,
	'╥': maskSingleLeft | maskSingleRight | maskDoubleBottom,

	'╧': maskDoubleLeft | maskDoubleRight | maskSingleTop,
	'╨': maskSingleLeft | maskSingleRight | maskDoubleTop,

	'╪': maskDoubleLeft | maskDoubleRight | maskSingleTop | maskSingleBottom,
	'╫': maskSingleLeft | maskSingleRight | maskDoubleTop | maskDoubleBottom,
}

var maskToChar = func() map[int]rune {
	result := make(map[int]rune, len(charToMask))
	for char, mask := range charToMask {
		result[mask] = char
	}
	return result
}()

// neighbors are the 4 characters around a cell.
type neighbors struct {
	left   rune
	right  rune
	top    rune
	bottom rune
}

func standardize(char rune) rune {
	if standardized, ok := standardizedChars[char]; ok {
		return standardized
	}
	return char
}

func isConnectable(char rune) bool {
	_, ok := charToMask[standardize(char)]
	return ok
}

func charMask(char rune, mask int) int {
	return charToMask[standardize(char)] & mask
}

// crossingChar decides the character where upper is drawn over lower.
// It returns false when no character fits the combined directions.
func crossingChar(upper rune, upperNeighbors neighbors, lower rune, lowerNeighbors neighbors) (rune, bool) {
	maskUpper := charMask(upper, maskCross)
	// Directions in the lower char, excluding the directions the upper char already has.
	maskLower := charMask(lower, excludeMask(maskUpper))

	// A neighbor counts when it has a stroke toward this cell.
	outerMask := 0
	if charMask(upperNeighbors.left, maskRight) > 0 || charMask(lowerNeighbors.left, maskRight) > 0 {
		outerMask |= maskLeft
	}
	if charMask(upperNeighbors.right, maskLeft) > 0 || charMask(lowerNeighbors.right, maskLeft) > 0 {
		outerMask |= maskRight
	}
	if charMask(upperNeighbors.top, maskBottom) > 0 || charMask(lowerNeighbors.top, maskBottom) > 0 {
		outerMask |= maskTop
	}
	if charMask(upperNeighbors.bottom, maskTop) > 0 || charMask(lowerNeighbors.bottom, maskTop) > 0 {
		outerMask |= maskBottom
	}

	innerMask := maskUpper | maskLower
	outerDirectionCount := bits.OnesCount(uint(outerMask)) / 3 // 3 is the number of line styles.

	mask := innerMask & outerMask
	if bits.OnesCount(uint(maskUpper)) > outerDirectionCount {
		// The upper char has more directions than its neighbors, for example a sole char
		// with not enough adjacent chars. Use the inner mask only, like ├ for a line anchor.
		mask = innerMask
	}

	char, ok := maskToChar[mask]
	return char, ok
}

// excludeMask returns a mask without any direction in mask, in all line styles.
// For example, │ (0000.0000.1100) gives 0011.0011.0011.
func excludeMask(mask int) int {
	// and with maskCross to remove the overflow bits.
	allDirectionsMask := ((mask << 8) | (mask << 4) | mask | (mask >> 4) | (mask >> 8)) & maskCross
	return maskCross ^ allDirectionsMask
}
