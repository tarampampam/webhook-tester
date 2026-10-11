package frontend

import "sync"

type lruEntry[K comparable, V any] struct {
	key        K
	val        V
	prev, next *lruEntry[K, V]
}

// lru is a thread-safe, fixed-capacity lru cache.
//
// Do not copy after first use.
type lru[K comparable, V any] struct {
	mu    sync.Mutex
	cap   int
	items map[K]*lruEntry[K, V]
	head  *lruEntry[K, V]
	tail  *lruEntry[K, V]
}

// newLRU creates a new lru cache with the given capacity.
func newLRU[K comparable, V any](cacheCap int) *lru[K, V] {
	if cacheCap <= 0 {
		panic("lru: capacity must be > 0")
	}

	c := &lru[K, V]{
		cap:   cacheCap,
		items: make(map[K]*lruEntry[K, V], cacheCap),
		head:  &lruEntry[K, V]{},
		tail:  &lruEntry[K, V]{},
	}
	c.head.next = c.tail
	c.tail.prev = c.head

	return c
}

// Get returns the value associated with key, or the zero value and false if not present.
func (c *lru[K, V]) Get(k K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[k]; ok {
		c.moveToFront(e)

		return e.val, true
	}

	var zero V

	return zero, false
}

// Set adds or updates an entry, evicting the lru entry when at capacity.
func (c *lru[K, V]) Set(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[k]; ok {
		e.val = v
		c.moveToFront(e)

		return
	}

	if len(c.items) == c.cap {
		entry := c.tail.prev
		c.unlink(entry)
		delete(c.items, entry.key)
	}

	e := &lruEntry[K, V]{key: k, val: v}
	c.items[k] = e
	c.link(e)
}

func (c *lru[K, V]) link(e *lruEntry[K, V]) {
	e.prev = c.head
	e.next = c.head.next
	e.prev.next, e.next.prev = e, e
}

func (c *lru[K, V]) unlink(e *lruEntry[K, V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev, e.next = nil, nil
}

func (c *lru[K, V]) moveToFront(e *lruEntry[K, V]) {
	if c.head.next == e {
		return
	}

	c.unlink(e)
	c.link(e)
}
