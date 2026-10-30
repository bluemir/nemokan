package uuid

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	id := Generate()
	assert.Regexp(t, regexp.MustCompile(`^02-[A-Za-z0-9_-]{27}$`), id)
	assert.NotEqual(t, id, Generate())
}
