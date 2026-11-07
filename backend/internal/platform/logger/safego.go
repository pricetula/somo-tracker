package logger

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

// SafeGo runs a function in a goroutine with panic recovery and error logging.
// The function receives a context that is cancelled when the parent context is cancelled.
// Panics are recovered, logged at Error level with stack trace, and the error is
// sent to the error channel if provided.
//
// Usage:
//
//	errCh := make(chan error, 1)
//	logger.SafeGo(ctx, func(ctx context.Context) error {
//	    // background work
//	    return nil
//	}, errCh)
//
//	select {
//	case err := <-errCh:
//	    // handle error
//	case <-ctx.Done():
//	    // cancelled
//	}
func SafeGo(ctx context.Context, fn func(context.Context) error, errCh chan<- error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err := &PanicError{Value: r, Stack: stack}
				slog.ErrorContext(ctx, "panic recovered",
					"panic", r,
					"stack", string(stack),
				)
				if errCh != nil {
					select {
					case errCh <- err:
					default:
					}
				}
			}
		}()

		// Run with parent context
		if err := fn(ctx); errCh != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()
}

// PanicError wraps a recovered panic value
type PanicError struct {
	Value interface{}
	Stack []byte
}

func (e *PanicError) Error() string {
	return "panic recovered"
}

// SafeGoWithTimeout runs fn with a timeout. If fn doesn't complete within timeout,
// the context is cancelled and an error is sent to errCh.
func SafeGoWithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error, errCh chan<- error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	SafeGo(ctx, func(ctx context.Context) error {
		err := fn(ctx)
		done <- err
		return nil
	}, nil)

	select {
	case err := <-done:
		if errCh != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	case <-ctx.Done():
		if errCh != nil {
			select {
			case errCh <- ctx.Err():
			default:
			}
		}
	}
}

// Group runs multiple functions concurrently, waits for all to complete,
// and returns the first error (if any). Panics are recovered and logged.
func Group(ctx context.Context, fns ...func(context.Context) error) error {
	if len(fns) == 0 {
		return nil
	}

	errCh := make(chan error, len(fns))
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	for _, fn := range fns {
		SafeGo(ctx, fn, errCh)
	}

	var firstErr error
	for i := 0; i < len(fns); i++ {
		select {
		case err := <-errCh:
			if err != nil && firstErr == nil {
				firstErr = err
				cancel() // cancel remaining
			}
		case <-ctx.Done():
			// context cancelled, drain remaining errors
			for j := i + 1; j < len(fns); j++ {
				select {
				case err := <-errCh:
					if err != nil && firstErr == nil {
						firstErr = err
					}
				case <-time.After(100 * time.Millisecond):
					// timeout waiting for goroutine
				}
			}
			return firstErr
		}
	}

	return firstErr
}
