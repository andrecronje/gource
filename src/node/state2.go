package node

import (
	"sync"
)

type nodeState2 struct {
	cond *sync.Cond
	lock sync.RWMutex
	wip  int

	state        NodeState
	getStateChan chan NodeState
	setStateChan chan NodeState
}

func newNodeState2() *nodeState2 {
	ns := &nodeState2{
		cond:         sync.NewCond(&sync.Mutex{}),
		getStateChan: make(chan NodeState),
		setStateChan: make(chan NodeState),
	}
	go ns.mtx()
	return ns
}

func (s *nodeState2) mtx() {
	for {
		select {
		case s.state = <-s.setStateChan:
		case s.getStateChan <- s.state:
		}
	}
}

func (s *nodeState2) goFunc(fu func()) {
	go func() {
		s.lock.Lock()
		s.wip++
		s.lock.Unlock()

		fu()

		s.lock.Lock()
		s.wip--
		s.lock.Unlock()

		s.cond.L.Lock()
		defer s.cond.L.Unlock()
		s.cond.Broadcast()
	}()
}

func (s *nodeState2) waitRoutines() {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	for {
		s.lock.RLock()
		wip := s.wip
		s.lock.RUnlock()

		if wip != 0 {
			s.cond.Wait()
			continue
		}
		break
	}
}

func (s *nodeState2) getState() NodeState {
	return <-s.getStateChan
}

func (s *nodeState2) setState(state NodeState) {
	s.setStateChan <- state
}
