package utils

import (
	"sync/atomic"
	"fmt"
)

type AtomicInt64 int64

func NewAtomicInt64(initVal int64) *AtomicInt64 {
	a := AtomicInt64(initVal)
	return &a
}

func(a *AtomicInt64)Get() int64{
	return int64(*a)
}

func(a *AtomicInt64)Set(newVal int64) {
	atomic.StoreInt64((*int64)(a), newVal)
}

func (a *AtomicInt64) CompareAndSet(expect, update int64) bool {
	return atomic.CompareAndSwapInt64((*int64)(a), expect, update)
}

func(a *AtomicInt64)GetAndSet(newVal int64)int64{
	for {
		cur := a.Get()
		if a.CompareAndSet(cur, newVal) {
			return cur
		}
	}
}

func (a *AtomicInt64) GetAndIncrement() int64 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}

}

// GetAndDecrement gets the old value and then decrement by 1, this operation
// performs atomically.
func (a *AtomicInt64) GetAndDecrement() int64 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetAndAdd gets the old value and then add by delta, this operation
// performs atomically.
func (a *AtomicInt64) GetAndAdd(delta int64) int64 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// IncrementAndGet increments the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicInt64) IncrementAndGet() int64 {
	for {
		current := a.Get()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// DecrementAndGet decrements the value by 1 and then gets the value, this
// operation performs atomically.
func (a *AtomicInt64) DecrementAndGet() int64 {
	for {
		current := a.Get()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// AddAndGet adds the value by delta and then gets the value, this operation
// performs atomically.
func (a *AtomicInt64) AddAndGet(delta int64) int64 {
	for {
		current := a.Get()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt64) String() string {
	return fmt.Sprintf("%d", a.Get())
}