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

// RunConcurrently runs tasks concurrently with number of goroutines bounded by bound.
// If there is a panic executing a single Task then it will capture the panic and capture it as an error
// which will then subsequently be returned from this function. It will not propagate the panic causing the app to exit.
func RunConcurrently(ctx context.Context, tasks []Task, bound int) error {
	semaphore := make(chan struct{}, bound)
	errCh := make(chan error, len(tasks))
	defer func() {
		close(semaphore)
		close(errCh)
	}()
	wg := sync.WaitGroup{}
	for _, task := range tasks {
		if err := waitTillTokenAvailable(ctx, semaphore); err != nil {
			klog.Errorf("error while waiting for token to run task. Err: %v", err)
			break
		}
		wg.Add(1)
		klog.Infof("Starting async execution of task %s", task.Name)
		go func(task Task) {
			defer capturePanicAsError(task.Name, errCh)
			defer wg.Done()
			err := task.Fn(ctx)
			if err != nil {
				errCh <- err
			}
			<-semaphore
		}(task)
	}

	wg.Wait()

	errs := make([]error, 0, len(tasks))
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func capturePanicAsError(name string, errCh chan<- error) {
	if v := recover(); v != nil {
		stack := debug.Stack()
		panicErr := fmt.Errorf("Task: %s execution panicked: %v\n, stack-trace: %s\n", name, v, stack)
		errCh <- panicErr
	}
}

func waitTillTokenAvailable(ctx context.Context, semaphore chan<- struct{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case semaphore <- struct{}{}:
			return nil
		}
	}
}
