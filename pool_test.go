package sync_test

import (
	"context"
	"github.com/kushsharma/go-sync"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

type Worker struct {
	// id is the worker's unique identifier.
	id int
}

func TestPool_Count(t *testing.T) {
	ctx := context.Background()
	t.Run("should reflect current count after initial bootstrap", func(t *testing.T) {
		itemPool := sync.NewPool[*Worker](
			sync.WithSize[*Worker](10),
			sync.WithBootstrapItems[*Worker](5),
		)
		itemPool.SetFactory(ctx, func() interface{} {
			return &Worker{id: rand.Intn(1000)}
		})
		assert.Equal(t, int32(5), itemPool.Count())
	})
}

func TestPool_Borrow(t *testing.T) {
	ctx := context.Background()
	t.Run("should reflect current count after borrow", func(t *testing.T) {
		itemPool := sync.NewPool[*Worker](
			sync.WithSize[*Worker](5),
		)
		itemPool.SetFactory(ctx, func() interface{} {
			return &Worker{id: rand.Intn(1000)}
		})
		assert.Equal(t, int32(0), itemPool.Count())

		worker1 := itemPool.Borrow(ctx)
		worker1Name := worker1.id
		worker2 := itemPool.Borrow(ctx)
		assert.Equal(t, int32(2), itemPool.Count())

		itemPool.ReturnItem(worker1)
		assert.Equal(t, int32(2), itemPool.Count())

		worker1 = itemPool.Borrow(ctx)
		assert.Equal(t, worker1Name, worker1.id)

		worker3 := itemPool.Borrow(ctx)
		worker4 := itemPool.Borrow(ctx)
		assert.Equal(t, int32(4), itemPool.Count())

		itemPool.ReturnItem(worker1)
		itemPool.ReturnItem(worker2)
		itemPool.ReturnItem(worker3)
		itemPool.ReturnItem(worker4)
	})
	t.Run("should block when max size is reached", func(t *testing.T) {
		itemPool := sync.NewPool[*Worker](
			sync.WithSize[*Worker](2),
		)
		itemPool.SetFactory(ctx, func() interface{} {
			return &Worker{id: rand.Intn(1000)}
		})
		worker1 := itemPool.Borrow(ctx)
		worker2 := itemPool.Borrow(ctx)
		go func() {
			time.Sleep(100 * time.Millisecond)
			itemPool.ReturnItem(worker1)
		}()
		timeBeforeRequest := time.Now()
		worker3 := itemPool.Borrow(ctx)
		timeAfterRequest := time.Now()
		if timeAfterRequest.Sub(timeBeforeRequest) < 100*time.Millisecond {
			assert.Fail(t, "should have blocked for 100ms or more before returning worker3")
		}
		runtime.GC()
		assert.Equal(t, int32(2), itemPool.Count())

		itemPool.ReturnItem(worker2)
		itemPool.ReturnItem(worker3)
	})
}
