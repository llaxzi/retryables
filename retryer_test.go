package retryables_test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/llaxzi/retryables"
	"github.com/stretchr/testify/assert"
	"log"
	"syscall"
	"testing"
	"time"
)

func ExampleRetryer_Retry() {

	retryer := retryables.NewRetryer(nil)
	retryer.SetDelay(1, 2)
	retryer.SetCount(3)
	retryer.SetConditionFunc(func(err error) bool {
		return errors.Is(err, syscall.EBUSY) // Retry if error is "file busy"
	})
	arg := 0

	var data int
	err := retryer.Retry(func() error {
		var err error
		data, err = someFunc(arg)
		return err
	})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)

	// Output: 1
}
func someFunc(count int) (int, error) {
	if count < 2 {
		return count + 1, syscall.EBUSY
	}
	return 1, nil
}

func TestRetryer_Retry(t *testing.T) {
	tests := []struct {
		name        string
		retryCount  int
		retryFunc   func() error
		expectErr   bool
		expectTries int
	}{
		{
			name:       "Success on first attempt",
			retryCount: 3,
			retryFunc: func() error {
				return nil // Успешный вызов сразу
			},
			expectErr:   false,
			expectTries: 1,
		},
		{
			name:       "Fail after max retries",
			retryCount: 3,
			retryFunc: func() error {
				return errors.New("permanent error") // Всегда возвращает ошибку
			},
			expectErr:   true,
			expectTries: 3,
		},
		{
			name:       "Success after 2 retries",
			retryCount: 3,
			retryFunc: func() func() error {
				attempts := 0
				return func() error {
					attempts++
					if attempts < 3 {
						return errors.New("temporary error") // Два раза ошибка
					}
					return nil // Третий раз успех
				}
			}(),
			expectErr:   false,
			expectTries: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			retryer := retryables.NewRetryer(nil)
			retryer.SetCount(test.retryCount)
			retryer.SetConditionFunc(func(err error) bool {
				return err != nil
			})
			attempts := 0
			err := retryer.Retry(func() error {
				attempts++
				return test.retryFunc()
			})
			assert.Equal(t, test.expectErr, err != nil)
			assert.Equal(t, test.expectTries, attempts)
		})
	}

}

func TestRetryer_Retry_OnSpecificError(t *testing.T) {
	retryer := retryables.NewRetryer(nil)
	retryer.SetCount(4)

	retryableErr := errors.New("retryable error")
	otherErr := errors.New("any other error")

	retryer.SetConditionFunc(func(err error) bool {
		return errors.Is(err, retryableErr)
	})

	attempts := 0
	err := retryer.Retry(func() error {
		attempts++
		if attempts < 3 {
			return retryableErr
		}
		return otherErr
	})
	assert.Equal(t, 3, attempts)
	assert.Error(t, err)
	assert.ErrorIs(t, err, otherErr)
}

func TestRetryer_Retry_Backoff(t *testing.T) {
	retryer := retryables.NewRetryer(nil)
	retryer.SetCount(3)
	retryer.SetDelay(10*time.Millisecond, 20*time.Millisecond)
	retryer.SetConditionFunc(func(err error) bool {
		return err != nil
	})

	attempts := 0
	start := time.Now()

	err := retryer.Retry(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("retryable error")
		}
		return nil
	})

	duration := time.Since(start).Milliseconds()

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	assert.GreaterOrEqual(t, duration, int64(40))
}

func TestRetryer_Retry_Logs(t *testing.T) {
	var logBuffer bytes.Buffer

	retryer := retryables.NewRetryer(&logBuffer)
	retryer.SetCount(3)
	retryer.SetDelay(10*time.Millisecond, 20*time.Millisecond)
	retryer.SetConditionFunc(func(err error) bool {
		return err != nil
	})

	attempts := 0
	err := retryer.Retry(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("some error")
		}
		return nil
	})

	assert.NoError(t, err)
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Attempt 1/3 failed")
	assert.Contains(t, logOutput, "Attempt 2/3 failed")

}
