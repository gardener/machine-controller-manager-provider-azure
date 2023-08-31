// Copyright 2023 SAP SE or an SAP affiliate company
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
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

// RunGroup is a runner for concurrently spawning multiple asynchronous tasks. If any task
// errors or panics then these are captured as errors.
type RunGroup struct {
	wg        sync.WaitGroup
	semaphore chan struct{}
	errCh     chan error
}

// NewRunGroup creates a new RunGroup.
func NewRunGroup(numTasks, bound int) *RunGroup {
	return &RunGroup{
		wg:        sync.WaitGroup{},
		semaphore: make(chan struct{}, bound),
		errCh:     make(chan error, numTasks),
	}
}

// RunConcurrently runs tasks concurrently with number of goroutines bounded by bound.
// If there is a panic executing a single Task then it will capture the panic and capture it as an error
// which will then subsequently be returned from this function. It will not propagate the panic causing the app to exit.
func RunConcurrently(ctx context.Context, tasks []Task, bound int) []error {
	rg := NewRunGroup(len(tasks), bound)
	defer rg.Close()

	for _, task := range tasks {
		rg.Trigger(ctx, task)
	}
	return rg.WaitAndCollectErrors()
}

// Trigger executes the task in a go-routine.
func (g *RunGroup) Trigger(ctx context.Context, task Task) {
	if err := g.waitTillTokenAvailable(ctx); err != nil {
		klog.Errorf("error while waiting for token to run task. Err: %v", err)
		g.errCh <- fmt.Errorf("context cancelled, could not schedule task %s : %w", task.Name, err)
		return
	}
	g.wg.Add(1)
	go func(task Task) {
		defer g.wg.Done()
		defer capturePanicAsError(task.Name, g.errCh)
		err := task.Fn(ctx)
		if err != nil {
			g.errCh <- err
		}
		<-g.semaphore
	}(task)
}

// WaitAndCollectErrors waits for all tasks to finish, collects and returns any errors.
func (g *RunGroup) WaitAndCollectErrors() []error {
	g.wg.Wait()
	close(g.errCh)

	var errs []error
	for err := range g.errCh {
		errs = append(errs, err)
	}
	return errs
}

// Close closes the RunGroup
func (g *RunGroup) Close() {
	close(g.semaphore)
}

func (g *RunGroup) waitTillTokenAvailable(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case g.semaphore <- struct{}{}:
			return nil
		}
	}
}

// capturePanicAsError recovers from a panic if there is one. Creates an error from it which contains the debug stack
// trace as well and pushes the error to the provided error channel.
func capturePanicAsError(name string, errCh chan<- error) {
	if v := recover(); v != nil {
		stack := debug.Stack()
		panicErr := fmt.Errorf("Task: %s execution panicked: %v\n, stack-trace: %s\n", name, v, stack)
		errCh <- panicErr
	}
}
