package vector_clock

import (
	"fmt"
	"sort"
)

type VectorClock struct {
	id int
	vector map[int]int
}

func InitVectorClock(me int) VectorClock {
	return VectorClock{id: me, vector: map[int]int{me: 0}}
}

func (vc VectorClock) ID() int {
	return vc.id
}

func (vc VectorClock) Get() map[int]int {
	return vc.vector
}

func (vc VectorClock) String() string {
	keys := make([]int, 0, 1)
	for k := range vc.vector {
		keys = append(keys, k)
	}
	
	s := "["
	sort.Ints(keys)
	for i, k := range keys {
		s = fmt.Sprintf("%s(P%d: %d)", s, k, vc.vector[k])
		if i < len(keys) - 1 {
			s = fmt.Sprintf("%s, ", s)
		}
	}
	return fmt.Sprintf("%s]", s)
}

func (vc *VectorClock) Increment() {
	vc.vector[vc.id] = vc.vector[vc.id] + 1
}

func (vc *VectorClock) Merge(other VectorClock) {
	for k, v := range other.vector {
		if vcv, valid := vc.vector[k]; valid {
			if vcv < v {
				vc.vector[k] = v
			}
		}else{
			vc.vector[k] = v
		}
	}
}