package collectorprofiles

import "testing"

func TestBoundedBufferEvictionIsFIFO(t *testing.T) {
	first := []byte(`{"first":1}`)
	second := []byte(`{"second":2}`)
	third := []byte(`{"third":3}`)
	max := len(second) + len(third)
	b := NewBoundedBuffer(max)
	b.Add(first)
	b.Add(second)
	b.Add(third)
	if b.Size() > max {
		t.Fatalf("size %d > max %d", b.Size(), max)
	}
	chunks := b.Snapshot()
	if len(chunks) != 2 || string(chunks[0]) != string(second) || string(chunks[1]) != string(third) {
		t.Fatalf("expected FIFO eviction keeping second+third, got %d chunks", len(chunks))
	}
}
