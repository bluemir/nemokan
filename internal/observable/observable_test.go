package observable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValue_Observe(t *testing.T) {
	value := New(1)

	var got []int
	cancel := value.Observe(func(v int) { got = append(got, v) })
	value.Set(2)
	value.Set(2)
	cancel()
	value.Set(3)

	assert.Equal(t, []int{1, 2, 2}, got)
	assert.Equal(t, 3, value.Value())
}

func TestValue_NotifyInOrder(t *testing.T) {
	value := New("")

	var got []string
	value.Observe(func(string) { got = append(got, "first") })
	value.Observe(func(string) { got = append(got, "second") })
	got = nil
	value.Set("x")

	assert.Equal(t, []string{"first", "second"}, got)
}

func TestValue_CancelWhileNotifying(t *testing.T) {
	value := New(0)

	var got []int
	var cancel func()
	cancel = value.Observe(func(v int) {
		got = append(got, v)
		if v == 1 {
			cancel()
		}
	})
	value.Set(1)
	value.Set(2)

	assert.Equal(t, []int{0, 1}, got)
}
