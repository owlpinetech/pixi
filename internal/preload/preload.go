package preload

import (
	"io"
	"sync/atomic"
)

type preloadResult[T any] struct {
	item T
	err  error
}

type Preloader[T any] struct {
	closed      atomic.Bool
	loadFunc    func(index int) (T, error)
	preloadSize int

	notifyLoadMore chan struct{}
	notifyLoaded   chan preloadResult[T]
}

func NewPreloader[T any](loadFunc func(index int) (T, error), preloadSize int) *Preloader[T] {
	return &Preloader[T]{
		loadFunc:       loadFunc,
		preloadSize:    preloadSize,
		notifyLoadMore: make(chan struct{}, preloadSize),
		notifyLoaded:   make(chan preloadResult[T], preloadSize*2),
	}
}

func (p *Preloader[T]) Start() {
	go p.loader()
}

func (p *Preloader[T]) Stop() {
	if p.closed.Load() {
		return
	}
	p.closed.Store(true)
	close(p.notifyLoadMore)
}

func (p *Preloader[T]) Notify() {
	if p.closed.Load() {
		return
	}
	p.notifyLoadMore <- struct{}{}
}

func (p *Preloader[T]) Next() (T, error) {
	item, ok := <-p.notifyLoaded
	if !ok {
		var zero T
		return zero, io.EOF
	}
	return item.item, item.err
}

func (p *Preloader[T]) loader() {
	defer close(p.notifyLoaded)
	loadIndex := 0
	for range p.notifyLoadMore {
		item, err := p.loadFunc(loadIndex)
		p.notifyLoaded <- preloadResult[T]{item: item, err: err}
		loadIndex += 1
	}
}
