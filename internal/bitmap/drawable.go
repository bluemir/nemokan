package bitmap

import "strings"

// Drawable has enough information to generate a bitmap of any size.
type Drawable interface {
	Bitmap(width, height int) *Bitmap
}

// CharDrawable fills the whole area with one character.
type CharDrawable struct {
	Char rune
}

func (d CharDrawable) Bitmap(width, height int) *Bitmap {
	builder := NewBuilder(width, height)
	builder.FillChar(d.Char)
	return builder.Bitmap()
}

// NinePatchDrawable scales a pattern by repeating the cells in its repeatable ranges.
type NinePatchDrawable struct {
	Pattern         Pattern
	HorizontalRange RepeatableRange
	VerticalRange   RepeatableRange
}

// NewNinePatchDrawable scales the whole pattern in both directions.
func NewNinePatchDrawable(pattern Pattern) NinePatchDrawable {
	return NinePatchDrawable{
		Pattern:         pattern,
		HorizontalRange: NewScaleRange(0, pattern.Width-1),
		VerticalRange:   NewScaleRange(0, pattern.Height-1),
	}
}

func (d NinePatchDrawable) Bitmap(width, height int) *Bitmap {
	builder := NewBuilder(width, height)
	rowIndexes := d.VerticalRange.Indexes(height, d.Pattern.Height)
	columnIndexes := d.HorizontalRange.Indexes(width, d.Pattern.Width)
	for row := range height {
		for column := range width {
			char := d.Pattern.char(rowIndexes[row], columnIndexes[column])
			// TODO: Think about a way to support direction chars to 9-patch drawable
			builder.Put(row, column, char, char)
		}
	}
	return builder.Bitmap()
}

type Pattern struct {
	Width  int
	Height int
	chars  []rune
}

// PatternFromText reads lines of text as a pattern. Spaces become Transparent.
func PatternFromText(text string) Pattern {
	lines := strings.Split(text, "\n")
	chars := []rune(strings.Join(lines, ""))
	for i, char := range chars {
		if char == ' ' {
			chars[i] = Transparent
		}
	}
	return Pattern{Width: len([]rune(lines[0])), Height: len(lines), chars: chars}
}

func (p Pattern) char(row, column int) rune {
	return p.chars[row*p.Width+column]
}

// RepeatableRange decides how the cells in [start, end] fill the extra space.
type RepeatableRange interface {
	// Indexes maps each of size cells to a pattern index in [0, patternSize).
	Indexes(size, patternSize int) []int
}

// ScaleRange stretches the range: `01` -> `00001111`.
type ScaleRange struct {
	start int
	end   int
}

func NewScaleRange(start, endInclusive int) ScaleRange {
	return ScaleRange{start: max(0, min(start, endInclusive)), end: max(start, endInclusive)}
}

func (r ScaleRange) Indexes(size, patternSize int) []int {
	return rangeIndexes(r.start, r.end, size, patternSize, func(index, minRepeatingIndex, maxRepeatingIndex, rangeSize int) int {
		repeatingSize := maxRepeatingIndex - minRepeatingIndex + 1
		return minRepeatingIndex + (index-minRepeatingIndex)*rangeSize/repeatingSize
	})
}

// RepeatRange repeats the range: `01` -> `01010101`.
type RepeatRange struct {
	start int
	end   int
}

func NewRepeatRange(start, endInclusive int) RepeatRange {
	return RepeatRange{start: max(0, min(start, endInclusive)), end: max(start, endInclusive)}
}

func (r RepeatRange) Indexes(size, patternSize int) []int {
	return rangeIndexes(r.start, r.end, size, patternSize, func(index, minRepeatingIndex, maxRepeatingIndex, rangeSize int) int {
		return minRepeatingIndex + (index-minRepeatingIndex)%rangeSize
	})
}

// rangeIndexes keeps the cells before start and after end as they are, and fills the middle with scaleIndex.
// If patternSize is smaller than end, the last pattern index is used as end.
func rangeIndexes(start, end, size, patternSize int, scaleIndex func(index, minRepeatingIndex, maxRepeatingIndex, rangeSize int) int) []int {
	adjustedEnd := min(patternSize-1, end)
	rangeSize := adjustedEnd - start + 1
	repeatingLeft := start
	repeatingRight := size - (patternSize - adjustedEnd)

	indexes := make([]int, size)
	for index := range indexes {
		switch {
		case index < repeatingLeft:
			indexes[index] = index
		case index > repeatingRight:
			indexes[index] = adjustedEnd + index - repeatingRight
		default:
			indexes[index] = scaleIndex(index, repeatingLeft, repeatingRight, rangeSize)
		}
	}
	return indexes
}
