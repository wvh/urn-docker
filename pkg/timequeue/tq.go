// Package timequeue provides a heap-based priority queue ordered by time value.
//
// It is based on the heap implementation in the standard library.
package timequeue

import (
	"container/heap"
	"sync"
	"time"
)

// An Item is something we manage in a priority queue.
type Item struct {
	value interface{}
	t     time.Time
	// index of the item in the heap, needed by update
	index int
}

// Items is a list of items in a time queue. This holds the lock-free inner portion of the priority queue.
// This code is made more complex by the fact we need locking for e.g. JSON introspection, and we need to be careful
// not to have already locked code calling a method that would try to obtain a lock again, causing a deadlock.
type Items []*Item

// Len is the number of items in the list. It implements the sort interface.
func (items Items) Len() int {
	return len(items)
}

// Less compares the items' time field so they can be sorted. It implements the sort interface.
func (items Items) Less(i, j int) bool {
	return items[i].t.Before(items[j].t)
}

// Swap switches two items during sorting. It implements the sort interface.
func (items Items) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
	items[i].index = i
	items[j].index = j
}

// Push adds an item to the heap. It implements the heap interface.
func (items *Items) Push(x interface{}) {
	n := len(*items)
	item := x.(*Item)
	item.index = n
	*items = append(*items, item)
}

// Pop returns and deletes the earliest (highest priority) time from the list. It implements the heap interface.
func (items *Items) Pop() interface{} {
	old := *items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*items = old[0 : n-1]
	return item
}

// update modifies the time (hence priority) and value of an item in the queue.
func (items *Items) update(item *Item, value interface{}, t time.Time) {
	item.value = value
	item.t = t
	heap.Fix(items, item.index)
}

// TimeQueue is a time-sorted priority queue using heap.Interface.
// It is safe to use this queue's methods from multiple Go routines.
type TimeQueue struct {
	items Items
	mu    sync.RWMutex
}

// Len returns the number of items in the queue.
func (tq *TimeQueue) Len() int {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return tq.items.Len()
}

// Init initialises the heap after insertion. After calling, the list will be sorted by time.
func (tq *TimeQueue) Init() {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	heap.Init(&tq.items)
}

// Timer is a pair of time and action to schedule in the time queue.
type Timer struct {
	When  time.Time
	Value interface{}
}

// FromList creates a new time queue from a list of time and action pairs.
func FromList(timers []Timer) *TimeQueue {
	tq := TimeQueue{}
	tq.items = make([]*Item, len(timers))

	i := 0
	for _, timer := range timers {
		tq.items[i] = &Item{
			value: timer.Value,
			t:     timer.When,
			index: i,
		}
		i++
	}

	heap.Init(&tq.items)
	return &tq
}

func (tq *TimeQueue) peek() *Item {
	if tq.items.Len() == 0 {
		return nil
	}
	return (tq.items)[0]
}

// Peek returns the first item in the queue without removing it. If the queue is empty, Peek returns nil.
func (tq *TimeQueue) Peek() *Item {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	return tq.peek()
}

// Push adds an item to the queue.
func (tq *TimeQueue) Push(item *Item) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	heap.Push(&tq.items, item)
}

// Pop returns and deletes the item with the highest priority (nearest time) from the queue.
func (tq *TimeQueue) Pop() *Item {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	return heap.Pop(&tq.items).(*Item)
}

// update modifies the priority and value of an Item in the queue. It will reorder the queue if needed.
func (tq *TimeQueue) update(item *Item, value interface{}, t time.Time) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	tq.items.update(item, value, t)
}

// Loader returns a set function to load initial data in bulk, and an init function to be called when all elements have been loaded.
// The init function will return a populated and initialised time queue ready for use. It is not safe to call set after init has executed.
// It is faster to initalise data this way than to call Schedule on an already active queue.
//
// For example:
//   set, init := Loader()
//   set(timeT, doitFunc)
//   set(timeT, doitFunc)
//   set(timeT, doitFunc)
//   queue := init()
func Loader() (func(time.Time, interface{}), func() *TimeQueue) {
	var initialised bool // or sync.Once?

	tq := &TimeQueue{}

	return func(t time.Time, value interface{}) {
			if initialised {
				panic("it is not safe to call set after init")
			}
			tq.items = append(tq.items, &Item{
				value: value,
				t:     t,
				index: len(tq.items),
			})
		}, func() *TimeQueue {
			tq.Init()
			initialised = true
			np := tq // copy pointer
			tq = nil // throw away original
			return np
		}
}

// Schedule adds a new time and action to the queue.
func (tq *TimeQueue) Schedule(t time.Time, value interface{}) {
	tq.Push(&Item{
		value: value,
		t:     t,
	})
}

// Walk loops over the items in the queue and calls the given function on each.
// When the function returns false, the iteration is stopped.
// The function is not allowed to change item arguments.
func (tq *TimeQueue) Walk(f func(*Item) bool) {
	tq.mu.RLock()
	defer tq.mu.RUnlock()

	for _, item := range tq.items {
		if !f(item) {
			break
		}
	}
}

// PopWhile loops over the items in the queue calling the provided function,
// calling Pop as long as the function returns true. Popped items are returned in a slice.
// The callback function is not supposed to change the value of any items given to it.
func (tq *TimeQueue) PopWhile(f func(*Item) bool) (popped []*Item) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for tq.items.Len() > 0 && f(tq.peek()) {
		popped = append(popped, heap.Pop(&tq.items).(*Item))
	}

	return popped
}

// Delete loops over all items in the queue, deleting any item for which the callback function returns true.
func (tq *TimeQueue) Delete(f func(*Item) bool) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	// filter backing array in place
	s := tq.items[:0]
	for _, item := range tq.items {
		if !f(item) {
			s = append(s, item)
		}
	}

	// don't leak
	for i := len(s); i < len(tq.items); i++ {
		tq.items[i] = nil
	}

	tq.items = s
	heap.Init(&tq.items)
}
