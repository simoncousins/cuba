package cuba

type Handle[I any] struct {
	pool  *Pool[I]
	item  I
	items []I
}

func (handle *Handle[I]) Item() I {
	return handle.item
}

func (handle *Handle[I]) Push(item I) {
	handle.items = append(handle.items, item)
}

func (handle *Handle[I]) Sync() {
	// PushAll can return PoolAbortedErr, but we deliberately ignore it
	// silently here.
	handle.pool.PushAll(handle.items)
	handle.items = handle.items[:0]
}
