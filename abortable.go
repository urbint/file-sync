package fsync

import (
	"errors"
	"sync"
)

// ErrAborted is returned on operations that are abortable when they are aborted
var ErrAborted = errors.New("Aborted")

// ErrNotRunning is returned on operations that are abortable when they are aborted
var ErrNotRunning = errors.New("Not running")

// ErrAlreadyRunning is returned on operations that are already running
var ErrAlreadyRunning = errors.New("Already running")

// A TickFn is a function signature for an abortable task
type TickFn func() (result interface{}, err error)

// Abortable is a concurrency-safe control structure around running abortable tasks
type Abortable struct {
	mu   sync.Mutex
	quit chan struct{}
}

// Run runs the abortable task with the specified tick function.
//
// It returns a channel that will return the result when it is done, and a channel that reports errors
func (a *Abortable) Run(tick TickFn) (<-chan interface{}, <-chan error) {
	result := make(chan interface{})
	errs := make(chan error)
	go func() {
		a.mu.Lock()
		if a.quit != nil {
			errs <- ErrAlreadyRunning
			return
		}
		a.quit = make(chan struct{})
		defer func() { a.quit = nil }()

		a.mu.Unlock()

		for {
			select {
			case <-a.quit:
				errs <- ErrAborted
				return
			default:
				val, err := tick()
				if val != nil {
					result <- val
					return
				} else if err != nil {
					errs <- err
					return
				}
			}
		}
	}()

	return result, errs
}

// Abort aborts the currently running task
//
// It returns an error if no task is running
func (a *Abortable) Abort() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.quit == nil {
		return ErrNotRunning
	}
	close(a.quit)
	return nil
}
