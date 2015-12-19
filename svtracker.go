package svtracker

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

type SvTracker struct {
	ExitCode int
	Term     chan struct{}
	init     bool
	initCh   chan struct{}
	initMu   *sync.Mutex
	wg       *sync.WaitGroup
	wgSize   int64
}

func New() *SvTracker {
	st := &SvTracker{
		ExitCode: 0,
		Term:     make(chan struct{}),
		init:     false,
		initCh:   make(chan struct{}, 1),
		initMu:   &sync.Mutex{},
		wg:       &sync.WaitGroup{},
		wgSize:   0,
	}
	return st
}

func (st *SvTracker) Add() {
	atomic.AddInt64(&st.wgSize, 1)
	st.wg.Add(1)
	if !st.init {
		st.initMu.Lock()
		if !st.init {
			st.initCh <- struct{}{}
			st.init = true
		}
		st.initMu.Unlock()
	}
}

func (st *SvTracker) Done() {
	atomic.AddInt64(&st.wgSize, -1)
	st.wg.Done()
}

func (st *SvTracker) Wait() {
	<-st.initCh
	st.wg.Wait()
}

func (st *SvTracker) Complete() {
	for true{
		st.Term <- struct{}{}
	}
}

func (st *SvTracker) HandleSignals() {
	kill := make(chan os.Signal, 2)
	signal.Notify(kill, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-st.Term:
			break
		case <-kill:
			st.ExitCode = 2
			st.Complete()
		}
	}()
}

func (st *SvTracker) Exit() {
	os.Exit(st.ExitCode)
}

func (st *SvTracker) WaitAndExit() {
	st.Wait()
	st.Exit()
}
