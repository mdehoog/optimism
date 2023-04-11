package batcher

import (
	"sync"
)

type Runner struct {
	concurrency uint64
	poll        func() (func(), error)
	changed     func(uint64)

	wg      sync.WaitGroup
	cond    sync.Cond
	running uint64
}

func NewRunner(concurrency uint64, poll func() (func(), error), changed func(uint64)) *Runner {
	return &Runner{
		concurrency: concurrency,
		poll:        poll,
		changed:     changed,
	}
}

func (s *Runner) Can() bool {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	return s.can()
}

func (s *Runner) can() bool {
	return s.concurrency <= 0 || s.running < s.concurrency
}

func (s *Runner) Wait() error {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	for !s.can() {
		s.cond.Wait()
	}
	return s.try()
}

func (s *Runner) Close() {
	s.wg.Wait()
}

func (s *Runner) Try() error {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	return s.try()
}

func (s *Runner) try() error {
	if !s.can() {
		return nil
	}
	job, err := s.poll()
	if err != nil {
		return err
	}
	if job == nil {
		return nil
	}

	s.running++
	s.changed(s.running)
	s.wg.Add(1)
	go func() {
		defer func() {
			s.cond.L.Lock()
			s.running--
			s.changed(s.running)
			s.wg.Done()
			s.cond.L.Unlock()
			s.cond.Broadcast()
		}()
		job()
	}()
	return nil
}
