package retryables_test

import (
	"errors"
	"fmt"
	"github.com/llaxzi/retryables"
	"log"
	"syscall"
)

func ExampleRetryer_Retry() {

	retryer := retryables.NewRetryer(nil)
	retryer.SetDelay(1, 2)
	retryer.SetCount(3)
	retryer.SetConditionFunc(func(err error) bool {
		if errors.Is(err, syscall.EBUSY) {
			return true // Retry if error is "file busy"
		}
		return false // Don't retry otherwise
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
