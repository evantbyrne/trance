package trance

type Page[T any] struct {
	Column      string
	Direction   SortDirection
	Error       error
	HasNext     bool
	HasPrevious bool
	First       *T
	FirstValue  any
	Last        *T
	LastValue   any
	Limit       uint64
	Stream      *WeaveListStreamer[T]
	TotalSize   uint64
	WeaveConfig WeaveConfig
}

func (page *Page[T]) Then(callback func(*Page[T]) error) *Page[T] {
	if page.Error == nil {
		page.Error = callback(page)
	}
	return page
}
