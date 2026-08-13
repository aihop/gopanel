package api

import (
	"sync"
	"time"
)

const codeExecutorStatusCacheTTL = time.Minute

var codeExecutorStatusCache = struct {
	sync.Mutex
	statuses  []codeExecutorStatus
	expiresAt time.Time
	loading   chan struct{}
}{}

func loadCodeExecutorStatuses() []codeExecutorStatus {
	now := time.Now()
	codeExecutorStatusCache.Lock()
	if now.Before(codeExecutorStatusCache.expiresAt) && len(codeExecutorStatusCache.statuses) > 0 {
		statuses := append([]codeExecutorStatus(nil), codeExecutorStatusCache.statuses...)
		codeExecutorStatusCache.Unlock()
		return statuses
	}
	if loading := codeExecutorStatusCache.loading; loading != nil {
		codeExecutorStatusCache.Unlock()
		<-loading
		return loadCodeExecutorStatuses()
	}
	loading := make(chan struct{})
	codeExecutorStatusCache.loading = loading
	codeExecutorStatusCache.Unlock()

	statuses := detectCodeExecutorStatuses()
	codeExecutorStatusCache.Lock()
	codeExecutorStatusCache.statuses = append([]codeExecutorStatus(nil), statuses...)
	codeExecutorStatusCache.expiresAt = time.Now().Add(codeExecutorStatusCacheTTL)
	codeExecutorStatusCache.loading = nil
	close(loading)
	codeExecutorStatusCache.Unlock()
	return statuses
}

func detectCodeExecutorStatuses() []codeExecutorStatus {
	statuses := make([]codeExecutorStatus, len(codeExecutorDefinitions))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(codeExecutorDefinitions))
	for index := range codeExecutorDefinitions {
		go func(index int) {
			defer waitGroup.Done()
			statuses[index] = detectCodeExecutor(codeExecutorDefinitions[index])
		}(index)
	}
	waitGroup.Wait()
	return statuses
}
