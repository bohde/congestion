package congestion

import "fmt"

const (
	slowStart = stage(iota)
	waiting
	increasing
	recovering
)

type stage int

func (p stage) String() string {
	switch p {
	case slowStart:
		return "slowStart"
	case waiting:
		return "waiting"
	case increasing:
		return "increasing"
	case recovering:
		return "recovering"
	}
	return fmt.Sprintf("stage(%d)", p)
}
