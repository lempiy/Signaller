package room

import (
	"fmt"
	"sync"
)

func NewKeyQueue() *KeyQueue {
	return &KeyQueue{
		mx:     sync.RWMutex{},
		awaits: make(map[string]chan<- Event),
	}
}

type KeyQueue struct {
	mx     sync.RWMutex
	awaits map[string]chan<- Event
}

func (q *KeyQueue) Get(key string) chan<- Event {
	q.mx.RLock()
	defer q.mx.RUnlock()
	return q.awaits[key]
}

func (q *KeyQueue) Delete(key string) {
	q.mx.Lock()
	defer q.mx.Unlock()
	delete(q.awaits, key)
}

func (q *KeyQueue) Set(key string, value chan<- Event) error {
	q.mx.Lock()
	defer q.mx.Unlock()
	if q.awaits[key] != nil {
		return fmt.Errorf("waiter with the key '%s' already awaits", key)
	}
	q.awaits[key] = value
	return nil
}
