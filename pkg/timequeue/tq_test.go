package timequeue

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type myTodo string

var items = []struct {
	when int
	todo myTodo
}{
	{when: -1, todo: "just a while ago"},
	{when: 0, todo: "right now"},
	{when: 20, todo: "far future"},
	{when: 10, todo: "plenty of time still"},
	{when: 1, todo: "real soon now"},
	{when: -10, todo: "long ago"},
	{when: -20, todo: "oldest"},
}

func setupQueue(now time.Time) *TimeQueue {
	set, init := Loader()
	for _, item := range items {
		set(now.Add(time.Duration(item.when)*time.Second), item.todo)
	}
	return init()
}

func TestTimeQueue(t *testing.T) {
	now := time.Now()

	/*
		for i, item := range tq.items {
			t.Logf("item %d: %+v", i, item)
		}
	*/

	t.Run("pop", func(t *testing.T) {
		tq := setupQueue(now)
		got := tq.Pop()

		val, ok := got.value.(myTodo)
		if !ok {
			t.Fatalf("queue returned wrong value type, want: %T, got: %T", myTodo(""), got.value)
		}
		if val != "oldest" {
			t.Errorf("wrong oldest element, want: %v, got: %v", "oldest", val)
		}

		if tq.Len() != len(items)-1 {
			t.Errorf("wrong queue length after pop, want: %d, got: %d", len(items)-1, tq.Len())
		}
	})

	t.Run("push", func(t *testing.T) {
		tq := setupQueue(now)
		tq.Push(&Item{
			value: myTodo("ages ago"),
			t:     now.Add(-20 * time.Second),
		})

		if tq.Len() != len(items)+1 {
			t.Errorf("wrong queue length after push, want: %d, got: %d", len(items)+1, tq.Len())
		}
	})

	t.Run("pop all", func(t *testing.T) {
		// hopelessly roundabout way to copy and sort
		sorter := make([]struct{ val, index int }, len(items))
		for i, item := range items {
			sorter[i].val, sorter[i].index = item.when, i
		}

		sort.Slice(sorter, func(i, j int) bool { return sorter[i].val < sorter[j].val })

		sorted := make([]myTodo, len(sorter))
		for i := range sorter {
			sorted[i] = items[sorter[i].index].todo
		}

		tq := setupQueue(now)

		got := make([]myTodo, 0, len(sorted))
		for tq.Len() > 0 {
			item := tq.Pop()
			got = append(got, item.value.(myTodo))
		}

		if tq.Len() != 0 {
			t.Errorf("queue not empty, want: %d, got: %d", 0, tq.Len())
		}

		if len(got) != len(sorted) {
			t.Errorf("unexpected length, want: %d, got: %d", len(sorted), len(got))
		}

		if !reflect.DeepEqual(sorted, got) {
			t.Errorf("result does not match sorted input, want: %q, got: %q", sorted, got)
		}
	})

	t.Run("walk function", func(t *testing.T) {
		tq := setupQueue(now)
		tq.Push(&Item{
			value: myTodo("waldo"),
			t:     now.Add(5 * time.Second),
		})

		invoked := 0
		found := 0
		tq.Walk(func(item *Item) bool {
			invoked++
			if item.value.(myTodo) != "waldo" {
				return true
			}
			found++
			return false
		})

		// "waldo" is somewhere in the middle timewise, so not at the first index
		if invoked < 2 {
			t.Errorf("walkfn invoked too few times, want: > %d, got: %d", 2, invoked)
		}

		if invoked >= len(items) {
			t.Errorf("walkfn invoked too many times, want: < %d, got: %d", len(items), invoked)
		}

		if found != 1 {
			t.Errorf("walkfn should have stopped after match, want: %d matches, got: %d matches", 1, found)
		}
	})

	t.Run("pop function", func(t *testing.T) {
		var wantExpired int

		for _, item := range items {
			if item.when <= 0 {
				wantExpired++
			}
		}
		wantLeft := len(items) - wantExpired

		tq := setupQueue(now)

		invoked := 0
		expired := 0
		makeExpired := func(now time.Time) func(*Item) bool {
			return func(item *Item) bool {
				invoked++
				// !After = Before || Equal
				if !item.t.After(now) {
					expired++
					return true
				}
				return false
			}
		}

		popped := tq.PopWhile(makeExpired(now))
		poppedFirst := popped[0].value.(myTodo)
		poppedLast := popped[len(popped)-1].value.(myTodo)

		if len(popped) != wantExpired {
			t.Errorf("function returned wrong number of expired items, want: %d, got: %d", wantExpired, expired)
		}

		if expired != len(popped) {
			t.Errorf("wrong count for expired condition, want: %d, got: %d", len(popped), expired)
		}

		if tq.Len() != wantLeft {
			t.Errorf("wrong number of non-expired items left in queue, want: %d, got: %d", wantLeft, tq.Len())
		}

		if poppedFirst != "oldest" {
			t.Errorf("wrong first element in popped list, want: %q, got: %q", "oldest", poppedFirst)
		}

		if poppedLast != "right now" {
			t.Errorf("wrong last element in popped list, want: %q, got: %q", "right now", poppedLast)
		}

		// invoked one more than expired (peek), except if all items are popped from the queue
		if invoked != expired+1 || invoked == len(items) {
			t.Errorf("pop function should have been invoked more than it popped, want: > %d, got: %d", expired, invoked)
		}
	})

	t.Run("delete function", func(t *testing.T) {
		tq := setupQueue(now)

		tq.Push(&Item{
			value: myTodo("waldo 1"),
			t:     now.Add(2 * time.Second),
		})
		tq.Push(&Item{
			value: myTodo("waldo 2"),
			t:     now.Add(2 * time.Second),
		})

		// double-check
		if tq.Len() != len(items)+2 {
			t.Errorf("queue should have two more items for this test, want: %d items, got: %d items", len(items)+2, tq.Len())
		}

		tq.Delete(func(item *Item) bool {
			if strings.HasPrefix(string(item.value.(myTodo)), "waldo") {
				return true
			}
			return false
		})

		if tq.Len() != len(items) {
			t.Errorf("delete function should have deleted two items, want: %d items, got: %d items", len(items), tq.Len())
		}
	})

	t.Run("update", func(t *testing.T) {
		tq := setupQueue(now)

		item := &Item{
			value: "temporary item",
			t:     now.Add(1 * time.Second),
		}
		tq.Push(item)

		newTime := now.Add(-100 * time.Second)

		tq.update(item, item.value, newTime)

		got := tq.Pop()
		if got.value != item.value {
			t.Errorf("wrong item at the head of the queue, want: %q, got: %q", item.value, got.value)
		}

		if !got.t.Equal(newTime) {
			t.Errorf("wrong time for item, want: %v, got: %v", newTime, got.t)
		}
	})

	t.Run("schedule", func(t *testing.T) {
		tq := setupQueue(now)

		itemValue := myTodo("this is the one")
		itemTime := now.Add(-100 * time.Second)

		tq.Schedule(itemTime, itemValue)

		got := tq.Pop()
		if got.value != itemValue {
			t.Errorf("wrong item at the head of the queue, want: %q, got: %q", itemValue, got.value)
		}

		if !got.t.Equal(itemTime) {
			t.Errorf("wrong time for item, want: %v, got: %v", itemTime, got.t)
		}
	})

	t.Run("peek", func(t *testing.T) {
		tq := setupQueue(now)

		got := tq.Peek()

		val, ok := got.value.(myTodo)
		if !ok {
			t.Fatalf("queue returned wrong value type, want: %T, got: %T", myTodo(""), got.value)
		}

		if val != "oldest" {
			t.Errorf("wrong oldest element, want: %v, got: %v", "oldest", val)
		}
	})

	t.Run("peek (empty queue)", func(t *testing.T) {
		tq := TimeQueue{}
		item := tq.Peek()

		if item != nil {
			t.Errorf("empty queue should return no items, want: %v, got: %v", nil, item)
		}
	})
}

func TestLoaderPanic(t *testing.T) {
	var panicked bool
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	set, init := Loader()
	_ = init()
	set(time.Now(), myTodo("boom!"))

	if !panicked {
		t.Error("using set() after init() should panic, but didn't")
	}
}

func TestFromList(t *testing.T) {
	now := time.Now()

	var timers = make([]Timer, 0, len(items))
	for _, item := range items {
		timers = append(timers, Timer{
			Value: item.todo,
			When:  now.Add(time.Duration(item.when) * time.Second),
		})
	}

	tq := FromList(timers)

	if tq.Len() != len(items) {
		t.Errorf("wrong queue length, want: %d, got: %d", len(items), tq.Len())
	}

	got := tq.Pop()

	val, ok := got.value.(myTodo)
	if !ok {
		t.Fatalf("queue returned wrong value type, want: %T, got: %T", myTodo(""), got.value)
	}

	if val != "oldest" {
		t.Errorf("wrong oldest element, want: %v, got: %v", "oldest", val)
	}
}

func BenchmarkTimeNow(b *testing.B) {
	b.Run("time.Now", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = time.Now()
		}
	})

	cached := func() func() time.Time {
		now := time.Now()
		return func() time.Time {
			return now
		}
	}()

	b.Run("cached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = cached()
		}
	})
}
