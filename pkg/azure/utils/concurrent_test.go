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
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestRunConcurrentlyWithAllSuccessfulTasks(t *testing.T) {
	tasks := []Task{
		createSuccessfulTaskWithDelay("task-1", 5*time.Millisecond),
		createSuccessfulTaskWithDelay("task-2", 15*time.Millisecond),
		createSuccessfulTaskWithDelay("task-3", 10*time.Millisecond),
	}
	g := NewWithT(t)
	g.Expect(RunConcurrently(context.Background(), tasks, len(tasks))).To(HaveLen(0))
}

func TestRunConcurrentlyWithOnePanickyTask(t *testing.T) {
	tasks := []Task{
		createSuccessfulTaskWithDelay("task-1", 5*time.Millisecond),
		createPanickyTaskWithDelay("panicky-task-2", 15*time.Millisecond),
		createSuccessfulTaskWithDelay("task-3", 10*time.Millisecond),
	}
	g := NewWithT(t)
	g.Expect(RunConcurrently(context.Background(), tasks, len(tasks))).To(HaveLen(1))
}

func TestRunConcurrentlyWithPanickyAndErringTasks(t *testing.T) {
	tasks := []Task{
		createSuccessfulTaskWithDelay("task-1", 5*time.Millisecond),
		createPanickyTaskWithDelay("panicky-task-2", 15*time.Millisecond),
		createSuccessfulTaskWithDelay("task-3", 10*time.Millisecond),
		createErringTaskWithDelay("erring-task-4", 50*time.Millisecond),
	}
	g := NewWithT(t)
	g.Expect(RunConcurrently(context.Background(), tasks, len(tasks))).To(HaveLen(2))
}

func createSuccessfulTaskWithDelay(name string, delay time.Duration) Task {
	return Task{
		Name: name,
		Fn: func(ctx context.Context) error {
			tick := time.NewTicker(delay)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-tick.C:
					return nil
				}
			}
		},
	}
}

func createPanickyTaskWithDelay(name string, delay time.Duration) Task {
	return Task{
		Name: name,
		Fn: func(ctx context.Context) error {
			tick := time.NewTicker(delay)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-tick.C:
					panic("i panicked")
				}
			}
		},
	}
}

func createErringTaskWithDelay(name string, delay time.Duration) Task {
	return Task{
		Name: name,
		Fn: func(ctx context.Context) error {
			tick := time.NewTicker(delay)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-tick.C:
					return errors.New("this task will never succeed")
				}
			}
		},
	}
}
