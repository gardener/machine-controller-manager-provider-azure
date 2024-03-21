// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
