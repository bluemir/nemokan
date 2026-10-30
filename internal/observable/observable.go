// Package observable notifies changes of a value to its observers.
// It replaces MonoSketch livedata. See ADR-0003.
package observable

import "slices"

type Value[T any] struct {
	value T
	// observers are notified in the order they were added.
	observers []*observer[T]
}

type observer[T any] struct {
	notify func(T)
}

func New[T any](initValue T) *Value[T] {
	return &Value[T]{value: initValue}
}

func (v *Value[T]) Value() T {
	return v.value
}

// Set stores value and notifies every observer, even if value is unchanged.
func (v *Value[T]) Set(value T) {
	v.value = value
	// Copy so an observer can cancel itself or others while being notified.
	for _, observer := range slices.Clone(v.observers) {
		observer.notify(value)
	}
}

// Observe calls notify with the current value now and on every Set.
// Call the returned function to stop observing.
func (v *Value[T]) Observe(notify func(T)) (cancel func()) {
	added := &observer[T]{notify: notify}
	v.observers = append(v.observers, added)
	notify(v.value)
	return func() {
		v.observers = slices.DeleteFunc(v.observers, func(o *observer[T]) bool { return o == added })
	}
}
