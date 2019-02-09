package congestion

import (
	"testing"
)

func TestPriority(t *testing.T) {
	cases := []struct {
		Priorities []int
		Expected   int
	}{
		{[]int{0, 1}, 1},
		{[]int{1, 0}, 1},
		{[]int{0, 2, 1}, 2},
	}

	for _, tc := range cases {
		q := newQueue(10)
		for _, p := range tc.Priorities {
			r := rendezvouz{priority: p}
			q.Push(&r)
		}
		actual := q.Pop().priority
		if actual != tc.Expected {
			t.Errorf("Priority %v = %d, expected %d", tc.Priorities, actual, tc.Expected)
		}
	}
}

func TestRemove(t *testing.T) {
	a := rendezvouz{priority: 0}
	b := rendezvouz{priority: 1}
	c := rendezvouz{priority: 2}

	q := newQueue(10)
	for _, r := range []*rendezvouz{&a, &b, &c} {
		q.Push(r)
	}

	q.Remove(&c)

	actual := q.Pop().priority

	if actual != b.priority {
		t.Errorf("Got %d, expected %d", actual, b.priority)
	}
}

func TestDropLast(t *testing.T) {
	a := rendezvouz{priority: 0, errChan: make(chan error, 1)}
	b := rendezvouz{priority: 1, errChan: make(chan error, 1)}
	c := rendezvouz{priority: 2, errChan: make(chan error, 1)}

	q := newQueue(2)

	for _, r := range []*rendezvouz{&a, &b, &c} {
		q.Push(r)
	}

	if q.Len() != 2 {
		t.Errorf("Got %d, expected %d", 2, q.Len())
	}

	dropped := <-a.errChan
	if dropped != Dropped {
		t.Errorf("Got %d, expected %d", dropped, Dropped)
	}

}

func BenchmarkQueue(b *testing.B) {
	q := newQueue(b.N)
	r := rendezvouz{}

	for i := 0; i < b.N; i++ {
		q.Push(&r)
	}
}

func BenchmarkQueueAllocs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newQueue(10)
	}
}
