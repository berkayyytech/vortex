package audit

import (
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	Message   string
}

type Engine struct {
	mu   sync.RWMutex
	Logs []LogEntry
}

var GlobalEngine = &Engine{
	Logs: []LogEntry{},
}

func (e *Engine) Append(msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Append to the front for "recent first" order
	e.Logs = append([]LogEntry{{Timestamp: time.Now(), Message: msg}}, e.Logs...)
}

func (e *Engine) GetRecent(n int) []LogEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var recent []LogEntry
	count := n
	if len(e.Logs) < count {
		count = len(e.Logs)
	}
	for i := 0; i < count; i++ {
		recent = append(recent, e.Logs[i])
	}
	return recent
}

func (e *Engine) GetAll() []LogEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Logs
}

func (e *Engine) Search(query string) []LogEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if query == "" {
		return e.Logs
	}
	var results []LogEntry
	lowerQuery := strings.ToLower(query)
	for _, l := range e.Logs {
		if strings.Contains(strings.ToLower(l.Message), lowerQuery) {
			results = append(results, l)
		}
	}
	return results
}
