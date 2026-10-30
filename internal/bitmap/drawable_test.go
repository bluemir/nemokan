package bitmap

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testPattern = PatternFromText(lines(
	"+-~+",
	"| •|",
	"|• |",
	"+~-+",
))

func lines(rows ...string) string {
	return strings.Join(rows, "\n")
}

func TestNinePatch_NoRanges(t *testing.T) {
	target := NewNinePatchDrawable(testPattern)
	assert.Equal(t, lines(
		"+++--~~~++",
		"+++--~~~++",
		"+++--~~~++",
		"|||  •••||",
		"|||  •••||",
		"|||••   ||",
		"|||••   ||",
		"|||••   ||",
		"+++~~---++",
		"+++~~---++",
	), target.Bitmap(10, 10).String())
}

func TestNinePatch_Range1(t *testing.T) {
	target := NinePatchDrawable{
		Pattern: PatternFromText(lines(
			"+-+",
			"| |",
			"+-+",
		)),
		HorizontalRange: NewRepeatRange(1, 1),
		VerticalRange:   NewRepeatRange(1, 1),
	}
	assert.Equal(t, lines(
		"+---+",
		"|   |",
		"|   |",
		"|   |",
		"+---+",
	), target.Bitmap(5, 5).String())
}

func TestNinePatch_ScaleRepeat(t *testing.T) {
	target := NinePatchDrawable{
		Pattern:         testPattern,
		HorizontalRange: NewScaleRange(1, 2),
		VerticalRange:   NewRepeatRange(1, 2),
	}
	assert.Equal(t, lines(
		"+----~~~~+",
		"|    ••••|",
		"|••••    |",
		"|    ••••|",
		"|••••    |",
		"|    ••••|",
		"|••••    |",
		"|    ••••|",
		"|••••    |",
		"+~~~~----+",
	), target.Bitmap(10, 10).String())
}

func TestNinePatch_RepeatScale(t *testing.T) {
	target := NinePatchDrawable{
		Pattern:         testPattern,
		HorizontalRange: NewRepeatRange(1, 2),
		VerticalRange:   NewScaleRange(1, 2),
	}
	assert.Equal(t, lines(
		"+-~-~-~-~+",
		"| • • • •|",
		"| • • • •|",
		"| • • • •|",
		"| • • • •|",
		"|• • • • |",
		"|• • • • |",
		"|• • • • |",
		"|• • • • |",
		"+~-~-~-~-+",
	), target.Bitmap(10, 10).String())
}

func TestScaleRange_Indexes(t *testing.T) {
	target := NewScaleRange(3, 5)
	assert.Equal(t, []int{0, 1, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 6, 7}, target.Indexes(16, 8))
	assert.Equal(t, []int{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4}, target.Indexes(16, 5))
}

func TestRepeatRange_Indexes(t *testing.T) {
	target := NewRepeatRange(3, 5)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 3, 4, 5, 3, 4, 5, 3, 4, 6, 7}, target.Indexes(16, 8))
	assert.Equal(t, []int{0, 1, 2, 3, 4, 3, 4, 3, 4, 3, 4, 3, 4, 3, 4, 3}, target.Indexes(16, 5))
}
