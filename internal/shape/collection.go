package shape

type identified interface {
	ID() string
}

// quickList finds items by id quickly while keeping the order they were added.
// Items can also be moved up, down, to the top or to the bottom.
type quickList[T identified] struct {
	head  *node[T]
	tail  *node[T]
	nodes map[string]*node[T]
}

type node[T any] struct {
	previous *node[T]
	next     *node[T]
	value    T
}

type moveAction int

const (
	moveUp moveAction = iota
	moveDown
	moveTop
	moveBottom
)

func newQuickList[T identified]() *quickList[T] {
	head := &node[T]{}
	tail := &node[T]{}
	head.next = tail
	tail.previous = head
	return &quickList[T]{head: head, tail: tail, nodes: map[string]*node[T]{}}
}

func (l *quickList[T]) size() int {
	return len(l.nodes)
}

func (l *quickList[T]) get(id string) (T, bool) {
	found, ok := l.nodes[id]
	if !ok {
		var zero T
		return zero, false
	}
	return found.value, true
}

func (l *quickList[T]) items() []T {
	result := make([]T, 0, len(l.nodes))
	for current := l.head.next; current != l.tail; current = current.next {
		result = append(result, current.value)
	}
	return result
}

func (l *quickList[T]) addFirst(item T) bool {
	return l.insertAfter(item, l.head)
}

func (l *quickList[T]) addLast(item T) bool {
	return l.insertAfter(item, l.tail.previous)
}

// addAfter adds item right after the item with afterID. It returns false if afterID is not found.
func (l *quickList[T]) addAfter(item T, afterID string) bool {
	previous, ok := l.nodes[afterID]
	if !ok {
		return false
	}
	return l.insertAfter(item, previous)
}

func (l *quickList[T]) insertAfter(item T, previous *node[T]) bool {
	if _, ok := l.nodes[item.ID()]; ok {
		return false
	}
	added := &node[T]{previous: previous, next: previous.next, value: item}
	previous.next.previous = added
	previous.next = added
	l.nodes[item.ID()] = added
	return true
}

func (l *quickList[T]) remove(id string) (T, bool) {
	found, ok := l.nodes[id]
	if !ok {
		var zero T
		return zero, false
	}
	delete(l.nodes, id)
	l.unlink(found)
	return found.value, true
}

func (l *quickList[T]) unlink(target *node[T]) {
	target.previous.next = target.next
	target.next.previous = target.previous
	target.next = nil
	target.previous = nil
}

func (l *quickList[T]) move(id string, action moveAction) bool {
	if l.size() < 2 {
		return false
	}
	target, ok := l.nodes[id]
	if !ok {
		return false
	}

	var newPrevious *node[T]
	switch action {
	case moveUp:
		if target != l.tail.previous {
			newPrevious = target.next
		}
	case moveTop:
		if target != l.tail.previous {
			newPrevious = l.tail.previous
		}
	case moveDown:
		if target != l.head.next {
			newPrevious = target.previous.previous
		}
	case moveBottom:
		if target != l.head.next {
			newPrevious = l.head
		}
	}
	if newPrevious == nil {
		return false
	}

	l.unlink(target)
	target.previous = newPrevious
	target.next = newPrevious.next
	newPrevious.next.previous = target
	newPrevious.next = target
	return true
}

// twoWayQuickMap can be queried by key and by value.
// A key has only one value, and a value can have many keys.
type twoWayQuickMap[K identified, V identified] struct {
	keyToValue  map[string]V
	valueToKeys map[string]map[string]K
}

func newTwoWayQuickMap[K identified, V identified]() *twoWayQuickMap[K, V] {
	return &twoWayQuickMap[K, V]{
		keyToValue:  map[string]V{},
		valueToKeys: map[string]map[string]K{},
	}
}

func (m *twoWayQuickMap[K, V]) set(key K, value V) {
	m.removeKey(key.ID())
	m.keyToValue[key.ID()] = value
	keys, ok := m.valueToKeys[value.ID()]
	if !ok {
		keys = map[string]K{}
		m.valueToKeys[value.ID()] = keys
	}
	keys[key.ID()] = key
}

func (m *twoWayQuickMap[K, V]) removeKey(keyID string) {
	value, ok := m.keyToValue[keyID]
	if !ok {
		return
	}
	delete(m.keyToValue, keyID)
	keys := m.valueToKeys[value.ID()]
	delete(keys, keyID)
	if len(keys) == 0 {
		delete(m.valueToKeys, value.ID())
	}
}

// removeValue removes the value and all of its keys.
func (m *twoWayQuickMap[K, V]) removeValue(valueID string) {
	keys, ok := m.valueToKeys[valueID]
	if !ok {
		return
	}
	delete(m.valueToKeys, valueID)
	for keyID := range keys {
		delete(m.keyToValue, keyID)
	}
}

func (m *twoWayQuickMap[K, V]) keys(valueID string) []K {
	result := make([]K, 0, len(m.valueToKeys[valueID]))
	for _, key := range m.valueToKeys[valueID] {
		result = append(result, key)
	}
	return result
}

func (m *twoWayQuickMap[K, V]) get(keyID string) (V, bool) {
	value, ok := m.keyToValue[keyID]
	return value, ok
}

func (m *twoWayQuickMap[K, V]) pairs(each func(key K, value V)) {
	for keyID, value := range m.keyToValue {
		each(m.valueToKeys[value.ID()][keyID], value)
	}
}
