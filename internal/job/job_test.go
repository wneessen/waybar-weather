// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package job

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

type testType struct {
	count     int
	completed bool
}

func TestNew(t *testing.T) {
	job := New(time.Millisecond*100, func(context.Context) {})
	if job == nil {
		t.Fatal("expected job to be non-nil")
	}
}

func TestJob_Start(t *testing.T) {
	t.Run("job succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			tester := &testType{}

			ctx, cancel := context.WithCancel(t.Context())
			context.AfterFunc(ctx, func() {
				tester.completed = true
			})

			testJob := New(time.Millisecond*100, tester.testFunc)
			go testJob.Start(ctx)

			synctest.Wait()
			if tester.completed {
				t.Fatal("expected job to not be completed before context was cancelled")
			}

			cancel()
			synctest.Wait()
			if !tester.completed {
				t.Fatal("expected job to be completed after context was cancelled")
			}
		})
	})
	t.Run("job ticker executes", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond*100)
			tester := &testType{}

			testJob := New(time.Millisecond*10, tester.testFunc)
			testJob.Start(ctx)

			synctest.Wait()
			cancel()
			if tester.count != 5 {
				t.Errorf("expected job to execute 5 times, got %d", tester.count)
			}
		})
		t.Run("nil job returns", func(t *testing.T) {
			tester := New(time.Millisecond*100, nil)
			tester.Start(t.Context())
		})
	})
}

func (t *testType) testFunc(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		if t.count >= 5 {
			return
		}
		t.count++
	}
}
