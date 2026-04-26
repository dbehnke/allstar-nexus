package main

import "sync"

type RingBuffer struct {
	mu     sync.Mutex
	events []*AMIEvent
	size   int
	count  int
	start  int
}

func newRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		events: make([]*AMIEvent, size),
		size:   size,
	}
}

func (rb *RingBuffer) Add(event *AMIEvent) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	idx := (rb.start + rb.count) % rb.size
	rb.events[idx] = event

	if rb.count < rb.size {
		rb.count++
	} else {
		rb.start = (rb.start + 1) % rb.size
	}
}

func (rb *RingBuffer) GetAll() []*AMIEvent {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	result := make([]*AMIEvent, 0, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.start + i) % rb.size
		result = append(result, rb.events[idx])
	}
	return result
}

func (rb *RingBuffer) Count() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}
