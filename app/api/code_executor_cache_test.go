package api

import (
	"sync"
	"testing"
	"time"
)

func resetCodeExecutorStatusCache() {
	codeExecutorStatusCache.Lock()
	codeExecutorStatusCache.statuses = nil
	codeExecutorStatusCache.expiresAt = time.Time{}
	codeExecutorStatusCache.loading = nil
	codeExecutorStatusCache.Unlock()
}

func TestCodeExecutorStatusCacheReturnsIndependentSlices(t *testing.T) {
	resetCodeExecutorStatusCache()
	t.Cleanup(resetCodeExecutorStatusCache)
	first := loadCodeExecutorStatuses()
	if len(first) == 0 {
		t.Fatal("executor status cache is empty")
	}
	first[0].Available = false
	second := loadCodeExecutorStatuses()
	if first[0].Available == second[0].Available {
		t.Fatal("caller mutation leaked into executor status cache")
	}
}

func TestCodeExecutorStatusCacheCoalescesConcurrentReads(t *testing.T) {
	resetCodeExecutorStatusCache()
	t.Cleanup(resetCodeExecutorStatusCache)
	const readers = 8
	results := make(chan []codeExecutorStatus, readers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(readers)
	for range readers {
		go func() {
			defer waitGroup.Done()
			results <- loadCodeExecutorStatuses()
		}()
	}
	waitGroup.Wait()
	close(results)
	for statuses := range results {
		if len(statuses) != len(codeExecutorDefinitions) {
			t.Fatalf("executor statuses = %d, want %d", len(statuses), len(codeExecutorDefinitions))
		}
	}
}
