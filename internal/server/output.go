package server

import (
	"strings"
	"sync"
)

// RingBuffer keeps last N lines of output
type RingBuffer struct {
	lines    []string
	capacity int
	head     int
	size     int
	mu       sync.Mutex
}

// NewRingBuffer creates a new ring buffer
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		lines:    make([]string, capacity),
		capacity: capacity,
	}
}

// Write adds a line to the buffer
func (b *RingBuffer) Write(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines[b.head] = line
	b.head = (b.head + 1) % b.capacity
	if b.size < b.capacity {
		b.size++
	}
}

// String returns all lines as a single string
func (b *RingBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	var result strings.Builder
	start := 0
	if b.size == b.capacity {
		start = b.head
	}
	for i := 0; i < b.size; i++ {
		idx := (start + i) % b.capacity
		result.WriteString(b.lines[idx])
		result.WriteString("\n")
	}
	return result.String()
}

// Lines returns all lines as a slice
func (b *RingBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]string, b.size)
	start := 0
	if b.size == b.capacity {
		start = b.head
	}
	for i := 0; i < b.size; i++ {
		idx := (start + i) % b.capacity
		result[i] = b.lines[idx]
	}
	return result
}

// Clear clears the buffer
func (b *RingBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.head = 0
	b.size = 0
}

// Size returns the current number of lines
func (b *RingBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}
