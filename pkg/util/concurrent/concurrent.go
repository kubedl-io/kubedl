package concurrent

import "sync"

func NewSemaphere(tickets int) *Semaphere {
	return &Semaphere{tickets: make(chan struct{}, tickets)}
}

type Semaphere struct {
	tickets chan struct{}
	wg      sync.WaitGroup
}

func (s *Semaphere) Acquire() {
	s.tickets <- struct{}{}
	s.wg.Add(1)
}

func (s *Semaphere) Release() {
	<-s.tickets
	s.wg.Done()
}

func (s *Semaphere) Wait() {
	s.wg.Wait()
	close(s.tickets)
}
