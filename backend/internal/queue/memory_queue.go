package queue

import (
	"container/heap"
	"sync"
	"time"

	"github.com/rudraprasaaad/task-scheduler/internal/models"
)

type PriorityQueue struct {
	items []*models.Task
	mutex sync.RWMutex
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*models.Task, 0),
	}
	heap.Init(pq)
	return pq
}

func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

func (pq *PriorityQueue) Less(i, j int) bool {
	if pq.items[i].Priority == pq.items[j].Priority {
		return pq.items[i].ScheduledAt.Before(pq.items[j].ScheduledAt)
	}

	return pq.items[i].Priority > pq.items[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	pq.items = append(pq.items, x.(*models.Task))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	pq.items = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Enqueue(task *models.Task) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	heap.Push(pq, task)
}

func (pq *PriorityQueue) Dequeue() *models.Task {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if pq.Len() == 0 {
		return nil
	}

	top := pq.items[0]

	if time.Now().Before(top.ScheduledAt) {
		return nil
	}

	return heap.Pop(pq).(*models.Task)
}

func (pq *PriorityQueue) Size() int {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	return pq.Len()
}
