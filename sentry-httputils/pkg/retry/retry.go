package retry

import (
	"fmt"
	"strings"
	"time"
)

// Function signature of retryable function
type RetryableFunc func() (string, int, error)
type RetryableFuncWithErrOnly func() error
type RetryableFuncWithValueAndErr func() (string,error)

var (
	DefaultAttempts      uint
	DefaultDelay         time.Duration
	DefaultMaxJitter     time.Duration
	DefaultOnRetry       = func(n uint, err error) { /* DefaultOnRetry */}
	DefaultRetryIf       = IsRecoverable
	DefaultDelayType     = CombineDelay(BackOffDelay, RandomDelay)
	DefaultLastErrorOnly bool
)

func SetConfig(defaultAttempts uint, defaultMaxJitter, defaultDelay time.Duration, defaultLastErrorOnly bool) {
	DefaultAttempts = defaultAttempts
	DefaultDelay = defaultDelay * time.Second
	DefaultMaxJitter = defaultMaxJitter * time.Second
	DefaultLastErrorOnly = defaultLastErrorOnly
}

func createErrorLog(config *Config) Error {
	var errorLog Error
	if !config.lastErrorOnly {
		errorLog = make(Error, config.attempts)
	} else {
		errorLog = make(Error, 1)
	}
	return errorLog
}

func Do(retryableFunc RetryableFunc, opts ...Option) (string, int, error) {
	var n uint

	//default
	config := &Config{
		attempts:      DefaultAttempts,
		delay:         DefaultDelay,
		maxJitter:     DefaultMaxJitter,
		onRetry:       DefaultOnRetry,
		retryIf:       DefaultRetryIf,
		delayType:     DefaultDelayType,
		lastErrorOnly: DefaultLastErrorOnly,
	}

	//apply opts
	for _, opt := range opts {
		opt(config)
	}

	errorLog := createErrorLog(config)

	resp := ""
	status := -1
	var err error
	err = nil
	lastErrIndex := n
	for n < config.attempts {
		resp, status, err = retryableFunc()

		if err != nil {
			errorLog[lastErrIndex] = unpackUnrecoverable(err)

			if !config.retryIf(err) {
				break
			}

			config.onRetry(n, err)

			// if this is last attempt - don't wait
			if n == config.attempts-1 {
				break
			}

			time.Sleep(getDelay(config, n))
		} else {
			return resp, status, nil
		}

		n++
		if !config.lastErrorOnly {
			lastErrIndex = n
		}
	}

	if config.lastErrorOnly {
		return resp, status, errorLog[lastErrIndex]
	}
	return resp, status, errorLog
}

func getDelay(config *Config, n uint) time.Duration {
	delayTime := config.delayType(n, config)
	if config.maxDelay > 0 && delayTime > config.maxDelay {
		delayTime = config.maxDelay
	}
	return delayTime
}

func Do_returnsOnlyError(retryableFunc RetryableFuncWithErrOnly, opts ...Option) error {
	var n uint

	//default
	config := &Config{
		attempts:      DefaultAttempts,
		delay:         DefaultDelay,
		maxJitter:     DefaultMaxJitter,
		onRetry:       DefaultOnRetry,
		retryIf:       DefaultRetryIf,
		delayType:     DefaultDelayType,
		lastErrorOnly: DefaultLastErrorOnly,
	}

	//apply opts
	for _, opt := range opts {
		opt(config)
	}

	errorLog := createErrorLog(config)

	lastErrIndex := n
	for n < config.attempts {
		err := retryableFunc()

		if err != nil {
			errorLog[lastErrIndex] = unpackUnrecoverable(err)

			if !config.retryIf(err) {
				break
			}

			config.onRetry(n, err)

			// if this is last attempt - don't wait
			if n == config.attempts-1 {
				break
			}

			time.Sleep(getDelay(config, n))
		} else {
			return nil
		}

		n++
		if !config.lastErrorOnly {
			lastErrIndex = n
		}
	}

	if config.lastErrorOnly {
		return errorLog[lastErrIndex]
	}
	return errorLog
}

func Do_returnsValueAndError(retryableFunc RetryableFuncWithValueAndErr, opts ...Option) (string,error) {
	var n uint

	//default
	config := &Config{
		attempts:      DefaultAttempts,
		delay:         DefaultDelay,
		maxJitter:     DefaultMaxJitter,
		onRetry:       DefaultOnRetry,
		retryIf:       DefaultRetryIf,
		delayType:     DefaultDelayType,
		lastErrorOnly: DefaultLastErrorOnly,
	}

	//apply opts
	for _, opt := range opts {
		opt(config)
	}
	errorLog := createErrorLog(config)

	lastErrIndex := n
	for n < config.attempts {
		value, err := retryableFunc()

		if err != nil {
			errorLog[lastErrIndex] = unpackUnrecoverable(err)

			if !config.retryIf(err) {
				break
			}

			config.onRetry(n, err)

			// if this is last attempt - don't wait
			if n == config.attempts-1 {
				break
			}

			time.Sleep(getDelay(config, n))
		} else {
			return value, nil
		}

		n++
		if !config.lastErrorOnly {
			lastErrIndex = n
		}
	}

	if config.lastErrorOnly {
		return "", errorLog[lastErrIndex]
	}
	return "", errorLog
}

// Error type represents list of errors in retry
type Error []error

// Error method return string representation of Error
// It is an implementation of error interface
func (e Error) Error() string {
	logWithNumber := make([]string, lenWithoutNil(e))
	for i, l := range e {
		if l != nil {
			logWithNumber[i] = fmt.Sprintf("#%d: %s", i+1, l.Error())
		}
	}

	return fmt.Sprintf("All attempts fail:\n%s", strings.Join(logWithNumber, "\n"))
}

func lenWithoutNil(e Error) (count int) {
	for _, v := range e {
		if v != nil {
			count++
		}
	}

	return
}

func (e Error) WrappedErrors() []error {
	return e
}

type unrecoverableError struct {
	error
}

// Unrecoverable wraps an error in `unrecoverableError` struct
func Unrecoverable(err error) error {
	return unrecoverableError{err}
}

// IsRecoverable checks if error is an instance of `unrecoverableError`
func IsRecoverable(err error) bool {
	_, isUnrecoverable := err.(unrecoverableError)
	return !isUnrecoverable
}

func unpackUnrecoverable(err error) error {
	if unrecoverable, isUnrecoverable := err.(unrecoverableError); isUnrecoverable {
		return unrecoverable.error
	}

	return err
}
