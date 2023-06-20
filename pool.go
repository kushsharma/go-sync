package sync

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// PoolOption configures the Pool
type PoolOption[T any] func(*Pool[T])

// WithBootstrapItems creates an initial number of ready-to-use items in the pool
func WithBootstrapItems[T any](c int) PoolOption[T] {
	return func(p *Pool[T]) {
		p.initial = c
	}
}

// WithSize limits the number of items in the pool
func WithSize[T any](l int) PoolOption[T] {
	return func(p *Pool[T]) {
		p.max = l
	}
}

// NewPool creates a new Pool.
func NewPool[T any](opts ...PoolOption[T]) *Pool[T] {
	pool := &Pool[T]{}
	for _, opt := range opts {
		opt(pool)
	}
	if pool.max < pool.initial {
		pool.max = pool.initial
	}
	if pool.max > 0 {
		pool.semMax = semaphore.NewWeighted(int64(pool.max))
	}

	return pool
}

// A Pool is a set of temporary objects that may be individually saved and
// retrieved.
//
// Any item stored in the Pool may be removed automatically at any time without
// notification. If the Pool holds the only reference when this happens, the
// item might be deallocated.
//
// A Pool is safe for use by multiple goroutines simultaneously.
//
// Pool's purpose is to cache allocated but unused items for later reuse,
// relieving pressure on the garbage collector. That is, it makes it easy to
// build efficient, thread-safe free lists. However, it is not suitable for all
// free lists.
//
// An appropriate use of a Pool is to manage a group of temporary items
// silently shared among and potentially reused by concurrent independent
// clients of a package. Pool provides a way to amortize allocation overhead
// across many clients.
//
// On the other hand, a free list maintained as part of a short-lived object is
// not a suitable use for a Pool, since the overhead does not amortize well in
// that scenario. It is more efficient to have such objects implement their own
// free list.
//
// A Pool must not be copied after first use.
type Pool[T any] struct {
	noCopy noCopy

	initial int
	max     int

	syncPool sync.Pool
	semMax   *semaphore.Weighted

	count atomic.Int32 // count keeps track of how many items are in the pool
}

// SetFactory specifies a function to generate an item when Borrow is called.
//
// Factory should only return pointer types
func (p *Pool[T]) SetFactory(ctx context.Context, factory func() any) {
	p.syncPool.New = func() any {
		newItem := factory()

		p.count.Add(1)
		runtime.SetFinalizer(newItem, func(newItem any) {
			p.count.Add(-1)
		})
		return newItem
	}

	if p.initial > 0 {
		// create initial number of items
		var items []T

		// create new items
		for i := 0; i < p.initial; i++ {
			items = append(items, p.Borrow(ctx))
		}
		// return new items
		for j := len(items) - 1; j >= 0; j-- {
			p.ReturnItem(items[j])
		}
		p.initial = 0
	}
}

// Borrow obtains an item from the pool.
// If the Max option is set, then this function
// will block until an item is returned back into the pool.
//
// After the item is no longer required, you must call
// Return on the item.
func (p *Pool[T]) Borrow(ctx context.Context) T {
	if p.semMax != nil {
		p.semMax.Acquire(ctx, 1)
	}
	return p.syncPool.Get().(T)
}

// ReturnItem returns an item back to the pool.
func (p *Pool[T]) ReturnItem(item T) {
	p.syncPool.Put(item)
	if p.semMax != nil {
		p.semMax.Release(1)
	}
}

// Count returns approximately the number of items in the pool (idle and in-use).
// If you want an accurate number, call runtime.GC() twice before calling Count (not recommended).
func (p *Pool[T]) Count() int32 {
	return p.count.Load()
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
