package retryables

import (
	"fmt"
	"io"
	"time"
)

type RetryableFunc func() error

func NewRetryer(logger io.Writer) *Retryer {
	if logger == nil {
		logger = io.Discard
	}
	return &Retryer{
		retryCount: 3,
		delay:      time.Second,
		increase:   2 * time.Second,
		retryConditionFunc: func(err error) bool {
			return err != nil
		},
		logger: logger,
	}
}

// A Retryer provides a mechanism for retrying operations with customizable settings.
// Warning: To ensure proper functionality, a new Retryer instance should be created whenever you need
// different retry settings (like different conditions or delays). However, if you have multiple operations
// that share the same retry settings, you can reuse a single Retryer instance.
type Retryer struct {
	retryConditionFunc func(error) bool
	retryCount         int
	delay              time.Duration
	increase           time.Duration
	logger             io.Writer
}

// Retry executes the given function with retries based on the configured settings.
// The number of attempts is set via SetCount, and the delay between attempts increases
// by the increment specified in SetDelay.
func (r *Retryer) Retry(retryFunc RetryableFunc) error {
	sleep := r.delay
	var err error
	for attempt := 0; attempt < r.retryCount; attempt++ {
		err = retryFunc()
		if err == nil {
			return nil
		}
		if !r.retryConditionFunc(err) {
			return err
		}
		_, _ = fmt.Fprintf(r.logger, "Attempt %d/%d failed: %v\n", attempt+1, r.retryCount, err)
		time.Sleep(sleep)
		sleep += r.increase
	}
	return err
}

// SetConditionFunc sets the condition function used to determine if an error should trigger a retry.
// This method is intended for initialization and is not thread-safe if modified dynamically at runtime.
func (r *Retryer) SetConditionFunc(retryConditionFunc func(error) bool) {
	r.retryConditionFunc = retryConditionFunc
}

// SetCount sets the number of attempts made by Retry() method.
// This method is intended for initialization and is not thread-safe if modified dynamically at runtime.
func (r *Retryer) SetCount(retryCount int) {
	r.retryCount = retryCount
}

// SetDelay sets the initial delay and increment for the backoff strategy used by Retry() method.
// This method is intended for initialization and is not thread-safe if modified dynamically at runtime.
func (r *Retryer) SetDelay(delay, increase time.Duration) {
	r.delay = delay
	r.increase = increase
}
