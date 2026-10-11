package storage

import "sync"

// pool is a generic, type-safe wrapper around [sync.Pool].
type pool[T any] struct{ p sync.Pool }

// newPool creates a Pool[T] that uses newFn to construct new instances of T when Get is called on an empty pool.
// newFn must not be nil.
func newPool[T any](newFn func() T) *pool[T] {
	return &pool[T]{p: sync.Pool{New: func() any { return newFn() }}}
}

// Get returns an instance of T from the pool. If the pool is empty, a new instance is constructed by calling newFn.
//
// The returned value's state is unspecified - callers must reset it before use.
func (p *pool[T]) Get() T { return p.p.Get().(T) } //nolint:errcheck,forcetypeassert

// Put returns v to the pool for later reuse via Get.
//
// After calling Put, v must not be used - another goroutine may retrieve it via Get at any time, and concurrent
// access would race.
func (p *pool[T]) Put(v T) { p.p.Put(v) }
