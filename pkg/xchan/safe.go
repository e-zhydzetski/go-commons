package xchan

import (
	"sync"
	"time"
)

type Opt func(safe *Safe)

func WithTestRetard(pauseDuration time.Duration) Opt {
	return func(safe *Safe) {
		safe.testRetarder = func() {
			time.Sleep(pauseDuration)
		}
	}
}

func MakeSafe(opts ...Opt) *Safe {
	s := &Safe{
		ch:           make(chan interface{}),
		closedCh:     make(chan struct{}),
		testRetarder: func() {}, // default without retarder
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type Safe struct {
	mx sync.RWMutex
	ch chan interface{}

	closeMx  sync.Mutex
	closedCh chan struct{}

	testRetarder func()
}

func (s *Safe) Ch() <-chan interface{} {
	return s.ch
}

func (s *Safe) Close() bool {
	if alreadyClosed := func() bool {
		s.closeMx.Lock()
		defer s.closeMx.Unlock()
		select {
		case <-s.closedCh:
			return true
		default:
			s.testRetarder() // for concurrent close test
			close(s.closedCh)
		}
		return false
	}(); alreadyClosed {
		return false
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
