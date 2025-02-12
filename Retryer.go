package retryables

import (
	"fmt"
	"io"
	"time"
)

type RetryableFunc func() error

// Retryer is an interface that provides a mechanism for retrying operations with customizable settings.
// Warning: To ensure proper functionality, a new Retryer instance should be created whenever you need
// different retry settings (like different conditions or delays). However, if you have multiple operations
// that share the same retry settings, you can reuse a single Retryer instance.
type Retryer interface {
	Retry(retryFunc RetryableFunc) error
	// SetConditionFunc sets the condition function used to determine if an error should trigger a retry.
	// This method is intended for initialization and is not thread-safe if modified dynamically at runtime.
	SetConditionFunc(retryConditionFunc func(error) bool)
	// SetCount sets the number of retry attempts by Retry() method.
	// This method is intended for initialization and is not thread-safe if modified dynamically at runtime.
	SetCount(retryCount int)
	// SetDelay sets the initial delay and increment for the backoff strategy used by Retry() method.
	// This method is intended for initialization and is not thread-safe if modified dynamically at runtime.
	SetDelay(delay, increase time.Duration)
}

func NewRetryer(logger io.Writer) Retryer {
	if logger == nil {
		logger = io.Discard
	}
	return &retryer{
		retryCount: 3,
		delay:      time.Second,
		increase:   2 * time.Second,
		retryConditionFunc: func(err error) bool {
			return err != nil
		},
		logger: logger,
	}

}

type retryer struct {
	retryConditionFunc func(error) bool
	retryCount         int
	delay              time.Duration
	increase           time.Duration
	logger             io.Writer
}

func (r *retryer) Retry(retryFunc RetryableFunc) error {
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
		sleep += r.increase
	}
	return err
}

func (r *retryer) SetConditionFunc(retryConditionFunc func(error) bool) {
	r.retryConditionFunc = retryConditionFunc
}

func (r *retryer) SetCount(retryCount int) {
	r.retryCount = retryCount
}

func (r *retryer) SetDelay(delay, increase time.Duration) {
	r.delay = delay
	r.increase = increase
}
