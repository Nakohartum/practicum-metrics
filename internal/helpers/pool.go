package helpers

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	pool []T
}

func New[T Resetter]() *Pool[T] {
	return &Pool[T]{
		pool: make([]T, 0),
	}
}

func (p *Pool[T]) Get() T {
	var zero T

	if len(p.pool) == 0 {
		return zero
	}

	lastIndex := len(p.pool) - 1
	item := p.pool[lastIndex]
	p.pool = p.pool[:lastIndex]

	return item
}

func (p *Pool[T]) Put(item T) {
	item.Reset()
	p.pool = append(p.pool, item)
}
