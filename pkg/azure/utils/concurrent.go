package utils

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"k8s.io/klog/v2"
)

// Task is a holder for a named function.
type Task struct {
	// Name is the name of the task
	Name string
	// Fn is the function which accepts a context and returns an error if there is one.
	// Implementations of Fn should handle context cancellation properly.
	Fn func(ctx context.Context) error
}

type runGroup struct {
	wg        sync.WaitGroup
	semaphore chan struct{}
}

// RunConcurrently runs tasks concurrently with number of goroutines bounded by bound.
// If there is a panic executing a single Task then it will capture the panic and capture it as an error
// which will then subsequently be returned from this function. It will not propagate the panic causing the app to exit.
func RunConcurrently(ctx context.Context, tasks []Task, bound int) error {
	runGrp := runGroup{
		wg:        sync.WaitGroup{},
		semaphore: make(chan struct{}, bound),
	}
	defer runGrp.Close()
	errCh := runGrp.triggerTasks(ctx, tasks)
	runGrp.wg.Wait()

	errs := make([]error, 0, len(tasks))
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (g *runGroup) triggerTasks(ctx context.Context, tasks []Task) <-chan error {
	errCh := make(chan error, len(tasks))
	defer close(errCh)
	for _, task := range tasks {
		if err := g.waitTillTokenAvailable(ctx); err != nil {
			klog.Errorf("error while waiting for token to run task. Err: %v", err)
			break
		}
		g.wg.Add(1)
		go func(task Task) {
			defer capturePanicAsError(task.Name, errCh)
			defer g.wg.Done()
			err := task.Fn(ctx)
			if err != nil {
				errCh <- err
			}
			<-g.semaphore
		}(task)
	}
	return errCh
}

func (g *runGroup) Close() {
	close(g.semaphore)
}

func capturePanicAsError(name string, errCh chan<- error) {
	if v := recover(); v != nil {
		stack := debug.Stack()
		panicErr := fmt.Errorf("Task: %s execution panicked: %v\n, stack-trace: %s\n", name, v, stack)
		errCh <- panicErr
	}
}

func (g *runGroup) waitTillTokenAvailable(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case g.semaphore <- struct{}{}:
			return nil
		}
	}
}
