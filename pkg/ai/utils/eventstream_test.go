package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEventStream_Basic(t *testing.T) {
	ctx := context.Background()
	stream := NewEventStream[int, string](ctx, 10)

	// Send some events
	go func() {
		for i := 0; i < 5; i++ {
			if err := stream.SendEvent(i); err != nil {
				t.Errorf("Failed to send event: %v", err)
			}
		}
		stream.SendResult("done")
	}()

	// Receive events
	count := 0
	for event := range stream.Events() {
		count++
		if event < 0 || event >= 5 {
			t.Errorf("Unexpected event value: %d", event)
		}
	}

	if count != 5 {
		t.Errorf("Expected 5 events, got %d", count)
	}

	// Get result
	result := <-stream.Result()
	if result != "done" {
		t.Errorf("Expected result 'done', got %s", result)
	}
}

func TestEventStream_Error(t *testing.T) {
	ctx := context.Background()
	stream := NewEventStream[int, string](ctx, 10)

	expectedErr := errors.New("test error")

	go func() {
		stream.SendEvent(1)
		stream.SendError(expectedErr)
	}()

	// Consume events
	for range stream.Events() {
	}

	if stream.Error() != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, stream.Error())
	}
}

func TestEventStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stream := NewEventStream[int, string](ctx, 10)

	// Cancel context
	cancel()

	// Try to send event after cancellation
	err := stream.SendEvent(1)
	if err == nil {
		t.Error("Expected error when sending to cancelled stream")
	}
}

func TestEventStream_Close(t *testing.T) {
	ctx := context.Background()
	stream := NewEventStream[int, string](ctx, 10)

	stream.Close()

	if !stream.IsClosed() {
		t.Error("Expected stream to be closed")
	}

	// Try to send event after close
	err := stream.SendEvent(1)
	if err == nil {
		t.Error("Expected error when sending to closed stream")
	}
}

func TestEventStream_Drain(t *testing.T) {
	ctx := context.Background()
	stream := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 0; i < 10; i++ {
			stream.SendEvent(i)
		}
		stream.SendResult("finished")
	}()

	result, err := stream.Drain()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "finished" {
		t.Errorf("Expected result 'finished', got %s", result)
	}
}

func TestEventStream_ForEach(t *testing.T) {
	ctx := context.Background()
	stream := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 0; i < 5; i++ {
			stream.SendEvent(i)
		}
		stream.SendResult("complete")
	}()

	sum := 0
	result, err := stream.ForEach(func(event int) error {
		sum += event
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "complete" {
		t.Errorf("Expected result 'complete', got %s", result)
	}
	if sum != 10 { // 0+1+2+3+4 = 10
		t.Errorf("Expected sum 10, got %d", sum)
	}
}

func TestMap(t *testing.T) {
	ctx := context.Background()
	source := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 0; i < 5; i++ {
			source.SendEvent(i)
		}
		source.SendResult("done")
	}()

	// Map int to string
	mapped := Map(source, func(i int) (string, error) {
		return string(rune('a' + i)), nil
	})

	// Collect results
	var results []string
	for event := range mapped.Events() {
		results = append(results, event)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	result := <-mapped.Result()
	if result != "done" {
		t.Errorf("Expected result 'done', got %s", result)
	}
}

func TestFilter(t *testing.T) {
	ctx := context.Background()
	source := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 0; i < 10; i++ {
			source.SendEvent(i)
		}
		source.SendResult("done")
	}()

	// Filter even numbers
	filtered := Filter(source, func(i int) bool {
		return i%2 == 0
	})

	// Collect results
	var results []int
	for event := range filtered.Events() {
		results = append(results, event)
	}

	if len(results) != 5 { // 0, 2, 4, 6, 8
		t.Errorf("Expected 5 results, got %d", len(results))
	}
}

func TestReduce(t *testing.T) {
	ctx := context.Background()
	source := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 1; i <= 5; i++ {
			source.SendEvent(i)
		}
		source.Close()
	}()

	// Sum all numbers
	sum, err := Reduce(source, 0, func(acc, val int) (int, error) {
		return acc + val, nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if sum != 15 { // 1+2+3+4+5 = 15
		t.Errorf("Expected sum 15, got %d", sum)
	}
}

func TestTee(t *testing.T) {
	ctx := context.Background()
	source := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 0; i < 5; i++ {
			source.SendEvent(i)
		}
		source.SendResult("done")
	}()

	targets := Tee(source, 2)

	if len(targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(targets))
	}

	// Collect from first target
	var results1 []int
	go func() {
		for event := range targets[0].Events() {
			results1 = append(results1, event)
		}
	}()

	// Collect from second target
	var results2 []int
	for event := range targets[1].Events() {
		results2 = append(results2, event)
	}

	time.Sleep(100 * time.Millisecond) // Wait for goroutine

	if len(results1) != 5 || len(results2) != 5 {
		t.Errorf("Expected 5 events in each target, got %d and %d", len(results1), len(results2))
	}
}

func TestMerge(t *testing.T) {
	ctx := context.Background()

	source1 := NewEventStream[int, string](ctx, 10)
	source2 := NewEventStream[int, string](ctx, 10)

	go func() {
		for i := 0; i < 3; i++ {
			source1.SendEvent(i)
		}
		source1.Close()
	}()

	go func() {
		for i := 3; i < 6; i++ {
			source2.SendEvent(i)
		}
		source2.Close()
	}()

	merged := Merge(ctx, source1, source2)

	var results []int
	for event := range merged.Events() {
		results = append(results, event)
	}

	if len(results) != 6 {
		t.Errorf("Expected 6 events, got %d", len(results))
	}
}
