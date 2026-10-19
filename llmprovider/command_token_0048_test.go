package llmprovider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 0048-MADR D4: the errors CommandToken.Token's doc names, beyond those
// TestCommandToken_Failures shows.

// TestCommandToken_CallerCancelIsItsContextError_0048: a caller whose context
// ends while its run is going gets its context's error, not ErrAuthFailure.
func TestCommandToken_CallerCancelIsItsContextError_0048(t *testing.T) {
	t.Parallel()
	c := NewCommandToken(helperArgv(t, "sleep")...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := c.Token(ctx)
		done <- err
	}()
	waitForInflight(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.inflight != nil
	})
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) || errors.Is(err, ErrAuthFailure) {
		t.Errorf("Token = %v; want context.Canceled, not ErrAuthFailure", err)
	}
}

// TestCommandToken_WaiterCancelIsItsContextError_0048: a caller waiting on
// another caller's run gets its own context's error when that context ends.
func TestCommandToken_WaiterCancelIsItsContextError_0048(t *testing.T) {
	t.Parallel()
	c := NewCommandToken(helperArgv(t, "sleep")...)
	joined := make(chan struct{}, 1)
	c.onJoin = func() { joined <- struct{}{} }
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()
	leaderDone := make(chan error, 1)
	go func() {
		_, err := c.Token(leaderCtx)
		leaderDone <- err
	}()
	waitForInflight(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.inflight != nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	waiterDone := make(chan error, 1)
	go func() {
		_, err := c.Token(ctx)
		waiterDone <- err
	}()
	<-joined
	if err := <-waiterDone; !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("waiter Token = %v; want its own context.DeadlineExceeded", err)
	}
	cancelLeader()
	<-leaderDone
}

// TestCommandToken_WaiterSharesTheRunsError_0048: a caller that joins a run
// gets that run's error: here its timeout, ErrAuthFailure.
func TestCommandToken_WaiterSharesTheRunsError_0048(t *testing.T) {
	t.Parallel()
	c := NewCommandToken(helperArgv(t, "sleep")...)
	c.Timeout = 300 * time.Millisecond
	joined := make(chan struct{}, 1)
	c.onJoin = func() { joined <- struct{}{} }
	leaderDone := make(chan error, 1)
	go func() {
		_, err := c.Token(context.Background())
		leaderDone <- err
	}()
	waitForInflight(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.inflight != nil
	})
	waiterDone := make(chan error, 1)
	go func() {
		_, err := c.Token(context.Background())
		waiterDone <- err
	}()
	<-joined
	leader, waiter := <-leaderDone, <-waiterDone
	if !errors.Is(leader, ErrAuthFailure) || !errors.Is(waiter, ErrAuthFailure) {
		t.Errorf("leader %v, waiter %v; want the run's timeout, ErrAuthFailure, for both", leader, waiter)
	}
}
