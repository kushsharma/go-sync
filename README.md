# go-sync

A thread safe sync object pool with maximum limit on pool size.

### Usage

```go
import "github.com/kushsharma/go-sync"

type Worker struct {
    // id is the worker's unique identifier.
    id int
}

itemPool := sync.New[*Worker](
    sync.WithSize[*Worker](5),
)
itemPool.SetFactory(ctx, func() interface{} {
    return &Worker{id: rand.Intn(1000)}
})

worker1 := itemPool.Borrow(ctx)
worker2 := itemPool.Borrow(ctx)

// Do something with worker1 and worker2

itemPool.ReturnItem(worker1)
itemPool.ReturnItem(worker1)
```