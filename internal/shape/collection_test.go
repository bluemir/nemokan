package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testItem struct {
	id string
}

func (i *testItem) ID() string { return i.id }

var (
	item0 = &testItem{id: "0"}
	item1 = &testItem{id: "1"}
	item2 = &testItem{id: "2"}
	item3 = &testItem{id: "3"}
)

func newTestQuickList(items ...*testItem) *quickList[*testItem] {
	list := newQuickList[*testItem]()
	for _, item := range items {
		list.addLast(item)
	}
	return list
}

func TestQuickList_Add(t *testing.T) {
	target := newQuickList[*testItem]()
	assert.Equal(t, 0, target.size())

	assert.True(t, target.addLast(item1))
	assert.False(t, target.addLast(item1))
	assert.Equal(t, 1, target.size())
	got, ok := target.get("1")
	assert.True(t, ok)
	assert.Equal(t, item1, got)

	target.addLast(item3)
	target.addAfter(item2, item1.ID())
	assert.Equal(t, []*testItem{item1, item2, item3}, target.items())

	assert.False(t, target.addAfter(item0, "unknown"))
}

func TestQuickList_Remove(t *testing.T) {
	target := newTestQuickList(item0, item1)

	_, ok := target.remove(item2.ID())
	assert.False(t, ok)

	removed, ok := target.remove("1")
	assert.True(t, ok)
	assert.Equal(t, item1, removed)
	assert.Equal(t, 1, target.size())
	_, ok = target.get("1")
	assert.False(t, ok)

	removed, _ = target.remove("0")
	assert.Equal(t, item0, removed)
	assert.Equal(t, 0, target.size())
}

func TestQuickList_Move(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		id       string
		action   moveAction
		expected []*testItem
	}{
		{"up", "1", moveUp, []*testItem{item0, item2, item1, item3}},
		{"down", "2", moveDown, []*testItem{item0, item2, item1, item3}},
		{"top", "1", moveTop, []*testItem{item0, item2, item3, item1}},
		{"bottom", "2", moveBottom, []*testItem{item2, item0, item1, item3}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target := newTestQuickList(item0, item1, item2, item3)
			target.move(testCase.id, testCase.action)
			assert.Equal(t, testCase.expected, target.items())
		})
	}
}

var (
	key1   = &testItem{id: "k1"}
	key2   = &testItem{id: "k2"}
	key3   = &testItem{id: "k3"}
	value1 = &testItem{id: "v1"}
	value2 = &testItem{id: "v2"}
)

func TestTwoWayQuickMap_Add(t *testing.T) {
	target := newTwoWayQuickMap[*testItem, *testItem]()

	target.set(key1, value1)
	got, _ := target.get("k1")
	assert.Equal(t, value1, got)
	assert.ElementsMatch(t, []*testItem{key1}, target.keys("v1"))
	_, ok := target.get(key2.ID())
	assert.False(t, ok)
	assert.Empty(t, target.keys("v2"))

	target.set(key2, value1)
	got, _ = target.get("k2")
	assert.Equal(t, value1, got)
	assert.ElementsMatch(t, []*testItem{key1, key2}, target.keys("v1"))
}

func TestTwoWayQuickMap_AddConflict(t *testing.T) {
	target := newTwoWayQuickMap[*testItem, *testItem]()
	target.set(key1, value1)
	target.set(key1, value1)
	target.set(key2, value1)
	target.set(key3, value2)

	target.set(key1, value2)
	got, _ := target.get(key1.ID())
	assert.Equal(t, value2, got)
	got, _ = target.get(key2.ID())
	assert.Equal(t, value1, got)
	got, _ = target.get(key3.ID())
	assert.Equal(t, value2, got)

	assert.ElementsMatch(t, []*testItem{key1, key3}, target.keys(value2.ID()))
	assert.ElementsMatch(t, []*testItem{key2}, target.keys(value1.ID()))
}

func TestTwoWayQuickMap_RemoveKey(t *testing.T) {
	target := newTwoWayQuickMap[*testItem, *testItem]()
	target.removeKey(key1.ID())

	target.set(key1, value1)
	target.set(key2, value2)
	target.removeKey(key1.ID())
	_, ok := target.get(key1.ID())
	assert.False(t, ok)
	assert.Empty(t, target.keys(value1.ID()))
	got, _ := target.get(key2.ID())
	assert.Equal(t, value2, got)
}

func TestTwoWayQuickMap_RemoveValue(t *testing.T) {
	target := newTwoWayQuickMap[*testItem, *testItem]()
	target.removeValue(value1.ID())

	target.set(key1, value1)
	target.set(key2, value1)
	target.set(key3, value2)

	target.removeValue(value1.ID())
	assert.Empty(t, target.keys(value1.ID()))
	_, ok := target.get(key1.ID())
	assert.False(t, ok)
	_, ok = target.get(key2.ID())
	assert.False(t, ok)
	got, _ := target.get(key3.ID())
	assert.Equal(t, value2, got)
	assert.ElementsMatch(t, []*testItem{key3}, target.keys(value2.ID()))
}
