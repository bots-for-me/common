package common

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type WaitChanResult chan struct{}
type WaitChan struct {
	waitChan chan WaitChanResult
	from     string
}
type WaitChans struct {
	items      []*WaitChan
	waitCalled bool
	mux        sync.Mutex
	parallel   bool
	waitChan   chan bool
}

type WaitChansParallel bool

func NewWaitChans(params ...interface{}) (w *WaitChans) {
	w = &WaitChans{
		waitChan: make(chan bool),
	}
	for _, param := range params {
		switch param.(type) {
		case WaitChansParallel:
			w.parallel = bool(param.(WaitChansParallel))
		}
	}
	return
}

func (w WaitChanResult) Done() {
	close(w)
}

func (w WaitChan) Wait() WaitChanResult {
	ch := <-w.waitChan
	return ch
}

func (w WaitChan) WaitAndDone() {
	w.Wait().Done()
}

func (w *WaitChans) Add() chan WaitChanResult {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		file = fmt.Sprintf("%v:%v", filepath.Base(file), line)
	}
	newChan := &WaitChan{
		waitChan: make(chan WaitChanResult, 1),
		from:     file,
	}
	// Log.Debug("WaitChans.Add() for %#v", file)
	if w.waitCalled {
		go func() {
			time.Sleep(time.Second)
			tmp := make(chan struct{})
			newChan.waitChan <- tmp
		}()
	} else {
		w.mux.Lock()
		w.items = append(w.items, newChan)
		w.mux.Unlock()
	}
	return newChan.waitChan
}

func (w *WaitChans) Remove(toRemove chan WaitChanResult) (found bool) {
	if toRemove == nil {
		return
	}
	w.mux.Lock()
	newLen := 0
	for i := range w.items {
		if w.items[i].waitChan == toRemove {
			close(w.items[i].waitChan)
			found = true
		} else {
			if i != newLen {
				w.items[newLen] = w.items[i]
			}
			newLen++
		}
	}
	w.items = w.items[0:newLen]
	w.mux.Unlock()
	return
}

func (w *WaitChans) Wait(args ...time.Duration) {
	// _, file, line, ok := runtime.Caller(1)
	// if ok {
	// 	file = fmt.Sprintf("%v:%v", filepath.Base(file), line)
	// }
	// Log.Info("WaitChans(%v) Wait...", file)
	// defer Log.Info("WaitChans(%v) Wait done", file)
	w.mux.Lock()
	defer w.mux.Unlock()
	if w.waitCalled {
		_, _ = <-w.waitChan
		return
	}
	w.waitCalled = true
	timeout := time.Second * 5
	if len(args) > 0 {
		timeout = args[0]
	}
	var wg sync.WaitGroup
	for {
		itemsLen := len(w.items)
		if itemsLen > 0 {
			ch := w.items[itemsLen-1]
			w.items = w.items[:itemsLen-1]
			if ch != nil {
				wg.Add(1)
				go func(ch *WaitChan) {
					tmp := make(WaitChanResult)
					// Log.Debug("WaitChans.Wait() wait %v for %#v", timeout, ch.from)
					ch.waitChan <- tmp
					timer := time.AfterFunc(timeout, func() {
						Log.Warn("WaitChans.Wait() timeout (%v, added in %#v)", timeout, ch.from)
						wg.Done()
					})
					// ts := time.Now()
					<-tmp
					// Log.Verbose("WaitChans.Wait() stopped in %v (added in %#v)", time.Since(ts), ch.from)
					if timer.Stop() {
						wg.Done()
					}
				}(ch)
				if !w.parallel {
					// Log.Verbose("WaitChans.Wait() wg.Wait() (for %#v)...", ch.from)
					wg.Wait()
					// Log.Verbose("WaitChans.Wait() wg.Wait() (for %#v) done", ch.from)
				}
			}
		} else {
			break
		}
	}
	if w.parallel {
		wg.Wait()
	}
	close(w.waitChan)
}
