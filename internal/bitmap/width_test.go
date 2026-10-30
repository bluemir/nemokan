package bitmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCharWidth(t *testing.T) {
	for char, expected := range map[rune]int{
		'a': 1, ' ': 1, '─': 1, '┼': 1,
		// Ambiguous
		'○': 1, '■': 1, '▶': 1,
		'한': 2, '漢': 2, 'あ': 2, 'Ａ': 2, '😀': 2,
		// Halfwidth katakana
		'ｱ': 1,
	} {
		assert.Equal(t, expected, CharWidth(char), string(char))
	}
	assert.Equal(t, 7, StringWidth("ab 한글"))
}

func TestBuilder_WideCharTailIsLeftOutOfString(t *testing.T) {
	builder := NewBuilder(5, 1)
	builder.Put(0, 0, '한', '한')
	builder.Put(0, 1, WideCharTail, WideCharTail)
	builder.Put(0, 2, 'a', 'a')
	assert.Equal(t, "한a  ", builder.Bitmap().String())
}
