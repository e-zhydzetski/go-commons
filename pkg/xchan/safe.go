package xchan

import (
	"sync"
)

func MakeSafe() *Safe {
	return &Safe{
		ch:       make(chan interface{}),
		closedCh: make(chan struct{}),
	}
}

type Safe struct {
	mx       sync.RWMutex
	ch       chan interface{}
	closedCh chan struct{}
}

func (s *Safe) Ch() <-chan interface{} {
	return s.ch
}

func (s *Safe) Close() bool {
	select {
	case <-s.closedCh:
		return false
	default:
		close(s.closedCh)
	}
	s.mx.Lock()
	close(s.ch)
	s.mx.Unlock()
	return true
}

func (s *Safe) Closed() <-chan struct{} {
	return s.closedCh
}

func (s *Safe) Send(msg interface{}) bool {
	select {
	case <-s.closedCh:
		return false
	default:
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	select {
	case s.ch <- msg:
		return true
	case <-s.closedCh:
		return false
	}
}
