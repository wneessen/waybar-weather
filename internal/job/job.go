// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package job

import (
	"context"
	"time"
)

// Job represents a scheduled task that runs at a fixed interval
// and never overlaps with itself (singleton mode).
type Job struct {
	interval time.Duration
	task     func(context.Context)
}

// New creates a new Job with the given interval and task.
func New(interval time.Duration, task func(context.Context)) *Job {
	return &Job{
		interval: interval,
		task:     task,
	}
}

// Start begins executing the job on the given context. It returns when the context is cancelled.
// It executes jobs in singleton mode, meaning if a tick fires while a previous run is still
// executing, that tick is skipped.
func (j *Job) Start(ctx context.Context) {
	if j.task == nil || j.interval <= 0 {
		return
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	// sem is a 1-slot semaphore that guards "is a run in progress?"
	sem := make(chan struct{}, 1)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Try to acquire the semaphore without blocking.
			select {
			case sem <- struct{}{}:
				go func() {
					defer func() { <-sem }()
					runCtx, cancel := context.WithCancel(ctx)
					defer cancel()
					j.task(runCtx)
				}()
			default:
			}
		}
	}
}
