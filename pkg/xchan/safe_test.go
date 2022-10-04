package xchan_test

import (
	"github.com/e-zhydzetski/go-commons/pkg/xchan"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestConcurrentClose(t *testing.T) {
	ch := xchan.MakeSafe(
		xchan.WithTestRetard(time.Millisecond),
	)
	const concurrent = 10

	var wg sync.WaitGroup
	wg.Add(concurrent)

	syncCh := make(chan struct{})
	for i := 0; i < concurrent; i++ {
		go func() {
			defer wg.Done()
			<-syncCh
			ch.Close()
		}()
	}
	close(syncCh)

	wg.Wait()
}

func TestReceiveAndClose(t *testing.T) {
	ch := xchan.MakeSafe()
	go func() {
		<-ch.Ch()
	}()
	runtime.Gosched()
	ch.Close()
}

func TestSendAndClose(t *testing.T) {
	ch := xchan.MakeSafe()
	go func() {
		ch.Send(struct{}{})
	}()
	runtime.Gosched()
	ch.Close()
}
