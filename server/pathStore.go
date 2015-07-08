package server

import (
	"sync"
)

type pathStore struct {
	m        *sync.Mutex
	timeMap  map[string]int64
	sequence []entry
}

func newPathStore(capacity int) *pathStore {
	p := new(pathStore)
	p.m = new(sync.Mutex)
	p.timeMap = make(map[string]int64, capacity)
	p.sequence = make([]entry, 0, capacity)
	return p
}

// @returns Number of clients remaining
func (ps *pathStore) add(address string, unixTime int64) int {
	ps.m.Lock()
	defer ps.m.Unlock()

	ps.timeMap[address] = unixTime
	ps.sequence = append(ps.sequence, entry{address, unixTime})
	return len(ps.timeMap)
}

// @returns Number of clients remaining
//func (ps *pathStore) delete(address string, unixTime int64) int {
//	ps.m.Lock()
//	defer ps.m.Unlock()
//
//	if ps.timeMap[address] == unixTime {
//		delete(ps.timeMap, address)
//	}
//	return len(ps.timeMap)
//}

// @returns Number of clients remaining
func (ps *pathStore) deleteBefore(unixTime int64) int {
	ps.m.Lock()
	defer ps.m.Unlock()

	pos := 0
L:
	for _, e := range ps.sequence {
		if e.time < unixTime {
			if ps.timeMap[e.address] < unixTime {
				delete(ps.timeMap, e.address)
			}
			pos++
		} else {
			break L
		}
	}
	ps.sequence = ps.sequence[pos:]

	//	if pos > 0 {
	//		log.Printf("pos: %d", pos)
	//	}

	return len(ps.timeMap)
}
