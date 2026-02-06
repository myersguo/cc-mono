package utils

import (
	"context"
	"fmt"
	"sync"
)

// EventStream is a generic event stream that supports streaming events
// of type T and producing a final result of type R
type EventStream[T any, R any] struct {
	ctx       context.Context
	cancel    context.CancelFunc
	events    chan T
	result    chan R
	err       error
	mu        sync.RWMutex
	closed    bool
	wg        sync.WaitGroup
	onceClose sync.Once
}

// NewEventStream creates a new event stream with the given buffer size
func NewEventStream[T any, R any](ctx context.Context, bufferSize int) *EventStream[T, R] {
	ctx, cancel := context.WithCancel(ctx)
	return &EventStream[T, R]{
		ctx:    ctx,
		cancel: cancel,
		events: make(chan T, bufferSize),
		result: make(chan R, 1),
	}
}

// Events returns the channel for receiving events
func (s *EventStream[T, R]) Events() <-chan T {
	return s.events
}

// Result returns the channel for receiving the final result
func (s *EventStream[T, R]) Result() <-chan R {
	return s.result
}

// Context returns the context associated with this stream
func (s *EventStream[T, R]) Context() context.Context {
	return s.ctx
}

// SendEvent sends an event to the stream
// Returns an error if the stream is closed or context is cancelled
func (s *EventStream[T, R]) SendEvent(event T) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("stream is closed")
	}
	s.mu.RUnlock()

	select {
	case s.events <- event:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

// SendResult sends the final result and closes the stream
// This should be called exactly once
func (s *EventStream[T, R]) SendResult(result R) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("stream is closed")
	}
	s.mu.RUnlock()

	select {
	case s.result <- result:
		s.Close()
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

// SendError sets an error and closes the stream
func (s *EventStream[T, R]) SendError(err error) {
	s.mu.Lock()
	s.err = err
	s.mu.Unlock()
	s.Close()
}

// Error returns the error if one occurred
func (s *EventStream[T, R]) Error() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.err
}

// Close closes the stream
// This is safe to call multiple times
func (s *EventStream[T, R]) Close() {
	s.onceClose.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		s.cancel()
		close(s.events)
		// Don't close result channel here - it might still have a value
	})
}

// IsClosed returns whether the stream is closed
func (s *EventStream[T, R]) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Wait blocks until the stream is closed
func (s *EventStream[T, R]) Wait() {
	s.wg.Wait()
}

// Drain consumes all events until the stream is closed
// Returns the final result and any error that occurred
func (s *EventStream[T, R]) Drain() (R, error) {
	var zero R

	// Drain all events
	for range s.events {
		// Discard events
	}

	// Check for errors first
	if err := s.Error(); err != nil {
		return zero, err
	}

	// Wait for result
	select {
	case result := <-s.result:
		return result, nil
	case <-s.ctx.Done():
		return zero, s.ctx.Err()
	}
}

// ForEach applies a function to each event in the stream
// Returns the final result and any error that occurred
func (s *EventStream[T, R]) ForEach(fn func(T) error) (R, error) {
	var zero R

	for {
		select {
		case event, ok := <-s.events:
			if !ok {
				// Events channel closed, wait for result
				goto WaitResult
			}
			if err := fn(event); err != nil {
				s.SendError(err)
				return zero, err
			}
		case <-s.ctx.Done():
			return zero, s.ctx.Err()
		}
	}

WaitResult:
	// Check for errors
	if err := s.Error(); err != nil {
		return zero, err
	}

	// Wait for result
	select {
	case result := <-s.result:
		return result, nil
	case <-s.ctx.Done():
		return zero, s.ctx.Err()
	}
}

// Map transforms events from one type to another
// Returns a new EventStream with the transformed events
func Map[T any, U any, R any](
	source *EventStream[T, R],
	transform func(T) (U, error),
) *EventStream[U, R] {
	target := NewEventStream[U, R](source.ctx, cap(source.events))

	go func() {
		defer target.Close()

		for {
			select {
			case event, ok := <-source.events:
				if !ok {
					// Source events closed, forward result
					select {
					case result := <-source.result:
						target.SendResult(result)
					case <-source.ctx.Done():
						target.SendError(source.ctx.Err())
					}
					return
				}

				transformed, err := transform(event)
				if err != nil {
					target.SendError(err)
					return
				}

				if err := target.SendEvent(transformed); err != nil {
					return
				}

			case <-source.ctx.Done():
				target.SendError(source.ctx.Err())
				return
			}
		}
	}()

	return target
}

// Filter creates a new EventStream that only contains events that pass the predicate
func Filter[T any, R any](
	source *EventStream[T, R],
	predicate func(T) bool,
) *EventStream[T, R] {
	target := NewEventStream[T, R](source.ctx, cap(source.events))

	go func() {
		defer target.Close()

		for {
			select {
			case event, ok := <-source.events:
				if !ok {
					// Source events closed, forward result
					select {
					case result := <-source.result:
						target.SendResult(result)
					case <-source.ctx.Done():
						target.SendError(source.ctx.Err())
					}
					return
				}

				if predicate(event) {
					if err := target.SendEvent(event); err != nil {
						return
					}
				}

			case <-source.ctx.Done():
				target.SendError(source.ctx.Err())
				return
			}
		}
	}()

	return target
}

// Reduce reduces a stream of events to a single value
func Reduce[T any, R any, A any](
	source *EventStream[T, R],
	initial A,
	reducer func(A, T) (A, error),
) (A, error) {
	accumulator := initial

	for {
		select {
		case event, ok := <-source.events:
			if !ok {
				// Events channel closed
				return accumulator, source.Error()
			}

			var err error
			accumulator, err = reducer(accumulator, event)
			if err != nil {
				source.SendError(err)
				return accumulator, err
			}

		case <-source.ctx.Done():
			return accumulator, source.ctx.Err()
		}
	}
}

// Tee splits a stream into multiple streams
// All target streams receive the same events
func Tee[T any, R any](
	source *EventStream[T, R],
	numTargets int,
) []*EventStream[T, R] {
	targets := make([]*EventStream[T, R], numTargets)
	for i := 0; i < numTargets; i++ {
		targets[i] = NewEventStream[T, R](source.ctx, cap(source.events))
	}

	go func() {
		defer func() {
			for _, target := range targets {
				target.Close()
			}
		}()

		for {
			select {
			case event, ok := <-source.events:
				if !ok {
					// Source events closed, forward result to all targets
					select {
					case result := <-source.result:
						for _, target := range targets {
							target.SendResult(result)
						}
					case <-source.ctx.Done():
						for _, target := range targets {
							target.SendError(source.ctx.Err())
						}
					}
					return
				}

				// Send event to all targets
				for _, target := range targets {
					if err := target.SendEvent(event); err != nil {
						return
					}
				}

			case <-source.ctx.Done():
				for _, target := range targets {
					target.SendError(source.ctx.Err())
				}
				return
			}
		}
	}()

	return targets
}

// Merge combines multiple streams into one
// Events from all source streams are sent to the target stream
func Merge[T any, R any](
	ctx context.Context,
	sources ...*EventStream[T, R],
) *EventStream[T, R] {
	target := NewEventStream[T, R](ctx, 100)

	var wg sync.WaitGroup
	wg.Add(len(sources))

	for _, source := range sources {
		go func(s *EventStream[T, R]) {
			defer wg.Done()

			for {
				select {
				case event, ok := <-s.events:
					if !ok {
						return
					}
					target.SendEvent(event)

				case <-s.ctx.Done():
					return
				}
			}
		}(source)
	}

	// Close target when all sources are done
	go func() {
		wg.Wait()
		target.Close()
	}()

	return target
}
