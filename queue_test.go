package congestion

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

func TestPriority(t *testing.T) {
	cases := []struct {
		Priorities []int
		Expected   int
	}{
		{[]int{0, 1}, 1},
		{[]int{1, 0}, 1},
		{[]int{0, 2, 1}, 2},
		{[]int{0, 2, 1, 3, 7, 8}, 8},
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

	r := q.Pop()

	if r.priority != b.priority {
		t.Errorf("Got %d, expected %d", r.priority, b.priority)
	}
}

func TestDropLast(t *testing.T) {
	cases := []int{2, 3, 4, 5, 6, 7, 8}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("%d", c), func(t *testing.T) {
			a := rendezvouz{priority: 0, errChan: make(chan error, 1)}

			q := newQueue(c)
			q.Push(&a)

			for i := 0; i < c; i++ {
				r := rendezvouz{priority: i + 1, errChan: make(chan error, 1)}
				q.Push(&r)
			}

			if q.Len() != c {
				t.Errorf("Got %d, expected %d", c, q.Len())
			}

			var dropped error

			select {
			case dropped = <-a.errChan:
			default:
			}

			if dropped != Dropped {
				t.Errorf("Got %d, expected %d", dropped, Dropped)
			}
		})
	}

}

func BenchmarkQueue(b *testing.B) {
	b.Run("newQueue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			newQueue(10)
		}

	})

	b.Run("Push", func(b *testing.B) {
		b.Run("Empty", func(b *testing.B) {
			q := newQueue(b.N)
			r := rendezvouz{}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				q.Push(&r)
			}

		})

		b.Run("Full", func(b *testing.B) {
			const (
				cap      = 10
				overflow = cap + 1
			)

			b.Run("Increasing", func(b *testing.B) {
				q := newQueue(cap)

				r := &rendezvouz{}
				for i := 0; i < cap; i++ {
					q.Push(r)
				}

				// Pre allocate enough rendezvouz instances to
				// so that we don't need to allocate in the loop
				rs := make([]rendezvouz, overflow)

				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					r := &rs[i%overflow]
					r.priority = i
					q.Push(r)
				}
			})

			b.Run("Decreasing", func(b *testing.B) {
				q := newQueue(cap)

				r := &rendezvouz{}
				for i := 0; i < cap; i++ {
					q.Push(r)
				}

				// Pre allocate enough rendezvouz instances to
				// so that we don't need to allocate in the loop
				rs := make([]rendezvouz, overflow)

				b.ResetTimer()

				for i := b.N; i > 0; i-- {
					r := &rs[i%overflow]
					r.priority = i
					q.Push(r)
				}
			})
		})
	})

	b.Run("Pop", func(b *testing.B) {
		q := newQueue(b.N)

		for i := 0; i < b.N; i++ {
			r := rendezvouz{}
			q.Push(&r)
		}

		b.ResetTimer()

		var out rendezvouz

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			out = q.Pop()
		}

		// To prevent optimization
		if out.priority != 0 {
		}

	})

}

type queueMachine struct {
	q *priorityQueue // queue being tested
	n int            // maximum queue size
}

// Init is an action for initializing  a queueMachine instance.
func (m *queueMachine) Init(t *rapid.T) {
	n := rapid.IntRange(1, 3).Draw(t, "n").(int)
	q := newQueue(n)
	m.q = &q
	m.n = n
}

// Model of Push
func (m *queueMachine) Push(t *rapid.T) {
	r := rendezvouz{
		priority: rapid.Int().Draw(t, "priority").(int),
		errChan:  make(chan error, 1),
	}

	m.q.Push(&r)
}

// Model of Remove
func (m *queueMachine) Remove(t *rapid.T) {
	if m.q.Empty() {
		t.Skip("empty")
	}

	r := (*m.q)[rapid.IntRange(0, m.q.Len()-1).Draw(t, "i").(int)]
	m.q.Remove(r)
}

// Model of Drop
func (m *queueMachine) Drop(t *rapid.T) {
	if m.q.Empty() {
		t.Skip("empty")
	}

	r := (*m.q)[rapid.IntRange(0, m.q.Len()-1).Draw(t, "i").(int)]
	r.Drop()
}

// Model of Signal
func (m *queueMachine) Pop(t *rapid.T) {
	if m.q.Empty() {
		t.Skip("empty")
	}

	r := m.q.Pop()
	r.Signal()
}

// validate that invariants hold
func (m *queueMachine) Check(t *rapid.T) {
	if m.q.Len() > m.q.Cap() {
		t.Fatalf("queue over capacity: %v vs expected %v", m.q.Len(), m.q.Cap())
	}

	for i, r := range *m.q {
		if r.index != i {
			t.Fatalf("illegal index: expected %d, got %+v ", i, r)
		}
	}

}

func TestPriorityQueue(t *testing.T) {
	t.Run("It should meet invariants", func(t *testing.T) {
		rapid.Check(t, rapid.Run(&queueMachine{}))
	})
}
