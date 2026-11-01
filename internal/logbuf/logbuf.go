// Package logbuf provides a concurrency-safe in-memory log ring with an
// optional append-only file mirror.
package logbuf

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Buffer keeps the most recent log lines.
type Buffer struct {
	mu    sync.Mutex
	lines []string
	max   int
	file  *os.File
}

// New creates a buffer retaining max lines.
func New(max int) *Buffer {
	if max <= 0 {
		max = 2000
	}
	return &Buffer{max: max}
}

// AttachFile mirrors future lines to path.
func (b *Buffer) AttachFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.file = f
	b.mu.Unlock()
	return nil
}

// Add appends a line with a timestamp.
func (b *Buffer) Add(line string) {
	stamped := time.Now().Format("15:04:05") + " " + line
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, stamped)
	if len(b.lines) > b.max {
		copy(b.lines, b.lines[len(b.lines)-b.max:])
		b.lines = b.lines[:b.max]
	}
	if b.file != nil {
		_, _ = fmt.Fprintln(b.file, stamped)
	}
}

// AddRaw appends a line without a timestamp.
func (b *Buffer) AddRaw(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, line)
	if len(b.lines) > b.max {
		copy(b.lines, b.lines[len(b.lines)-b.max:])
		b.lines = b.lines[:b.max]
	}
	if b.file != nil {
		_, _ = fmt.Fprintln(b.file, line)
	}
}

// Lines returns a copy of all buffered lines.
func (b *Buffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.lines...)
}

// Tail returns up to n trailing lines.
func (b *Buffer) Tail(n int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n > len(b.lines) {
		n = len(b.lines)
	}
	if n <= 0 {
		return nil
	}
	return append([]string(nil), b.lines[len(b.lines)-n:]...)
}

// Len returns the number of buffered lines.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.lines)
}

// Clear empties the in-memory buffer.
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = nil
}

// Close closes the mirrored file, if any.
func (b *Buffer) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.file != nil {
		_ = b.file.Close()
		b.file = nil
	}
}
