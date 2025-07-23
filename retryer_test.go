package retryables_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/llaxzi/retryables/v3"
)

func ExampleRetryer_Retry() {

	logger, _ := zap.NewProduction()

	retryer := retryables.NewRetryer(zap.NewStdLog(logger).Writer())
	retryer.SetDelay(1*time.Second, 5*time.Second)
	retryer.SetCount(3)
	retryer.SetConditionFunc(func(err error) bool {
		return errors.Is(err, syscall.EBUSY) // Retry if error is "file busy"
	})

	var data int
	err := retryer.Retry(context.Background(), func() error {
		var err error
		data, err = someFunc(data)
		return err
	})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)

	// Output: 2
}
func someFunc(count int) (int, error) {
	if count < 2 {
		return count + 1, syscall.EBUSY
	}
	return count, nil
}

func TestRetryer_Retry(t *testing.T) {
	excededCtx, _ := context.WithDeadline(context.Background(), time.Now())

	tests := []struct {
		name        string
		retryCount  int
		ctx         context.Context
		retryFunc   func() error
		expectErr   bool
		expectTries int
	}{
		{
			name:       "Success on first attempt",
			retryCount: 3,
			ctx:        context.Background(),
			retryFunc: func() error {
				return nil // Успешный вызов сразу
			},
			expectErr:   false,
			expectTries: 1,
		},
		{
			name:       "Fail after max retries",
			retryCount: 3,
			ctx:        context.Background(),
			retryFunc: func() error {
				return errors.New("permanent error") // Всегда возвращает ошибку
			},
			expectErr:   true,
			expectTries: 3,
		},
		{
			name:       "Success after 2 retries",
			retryCount: 3,
			ctx:        context.Background(),
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

		{
			name:       "Ctx Done",
			retryCount: 3,
			ctx:        excededCtx,
			retryFunc: func() error {
				return nil
			},
			expectErr:   true,
			expectTries: 0,
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
			err := retryer.Retry(test.ctx, func() error {
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
	err := retryer.Retry(context.Background(), func() error {
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
	rand.Seed(42) // фиксируем seed

	retryer := retryables.NewRetryer(nil)
	retryer.SetCount(3)
	retryer.SetDelay(20*time.Millisecond, 100*time.Millisecond)
	retryer.SetConditionFunc(func(err error) bool { return err != nil })

	attempts := 0
	start := time.Now()

	err := retryer.Retry(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("retryable error")
		}
		return nil
	})

	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)

	// Ожидаемый диапазон задержек
	// Backoff: attempt 0 -> ~20ms jitter [0,20), attempt 1 -> ~40ms jitter [0,40)
	expectedMin := 0 * time.Millisecond                                            // в худшем случае jitter = 0
	expectedMax := 20*time.Millisecond + 40*time.Millisecond + 10*time.Millisecond // +запас
	assert.GreaterOrEqual(t, duration, expectedMin)
	assert.LessOrEqual(t, duration, expectedMax)
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
	err := retryer.Retry(context.Background(), func() error {
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
