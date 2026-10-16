package tmux

import (
	"sync"
	"testing"
)

func TestTrackExcerptStability(t *testing.T) {
	m := NewManager("")

	if m.trackExcerpt("$1", "prompt>") {
		t.Fatal("first sighting should not be stable")
	}
	if m.trackExcerpt("$1", "prompt>") {
		t.Fatal("one repeat should not be stable yet")
	}
	if !m.trackExcerpt("$1", "prompt>") {
		t.Fatal("two repeats should be stable")
	}
	if m.trackExcerpt("$1", "new output") {
		t.Fatal("changed excerpt should reset stability")
	}
	if m.trackExcerpt("$2", "") || m.trackExcerpt("$2", "") || m.trackExcerpt("$2", "") {
		t.Fatal("empty excerpt should never be stable")
	}
}

// The poller tracks excerpts while removeSession can run from the watcher or
// HTTP handlers; run both concurrently under -race.
func TestTrackExcerptConcurrentWithRemove(t *testing.T) {
	m := NewManager("")
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			m.trackExcerpt("$1", "same")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			m.removeSession("$1")
		}
	}()
	wg.Wait()
}
