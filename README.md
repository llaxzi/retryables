# retryables
```retryables``` is a Go package for retrying operations with configurable attempts, delays, and retry conditions. It helps handle transient failures by automatically retrying failed operations.

## Installation

```sh
go get github.com/llaxzi/retryables
```
## Quick Start
```sh
retryer := retryables.NewRetryer(os.Stdout) // Retryer with logger
retryer.SetDelay(1, 2)
retryer.SetCount(3) // Make 3 attempts
retryer.SetConditionFunc(func(err error) bool {
  return errors.Is(err, syscall.EBUSY) // Retry if error is "file busy"
	})
// Usage
var data int
err := retryer.Retry(func() error {
  var err error
  data, err = someFunc(arg)
  return err
})
```
