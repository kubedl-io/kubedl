package concurrent

import "sync"

func NewSemaphore(tickets int) *Semaphore {
	return &Semaphore{tickets: make(chan struct{}, tickets)}
}

type Semaphore struct {
	tickets chan struct{}
	wg      sync.WaitGroup
}

func (s *Semaphore) Acquire() {
	s.tickets <- struct{}{}
	s.wg.Add(1)
}

func (s *Semaphore) Release() {
	<-s.tickets
	s.wg.Done()
}

func (s *Semaphore) Wait() {
	s.wg.Wait()
	close(s.tickets)
}
