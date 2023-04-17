package retry

import "time"

var defaultOptions = RetryOptions{
	maxAttempts:          60,
	delayBetweenAttempts: 1 * time.Second,
	logAttempts:          true,
}

type RetryOptions struct {
	maxAttempts          int
	delayBetweenAttempts time.Duration
	logAttempts          bool
}

func Options() RetryOptions {
	return defaultOptions
}

func (o RetryOptions) MaxAttempts(maxAttempts int) RetryOptions {
	o.maxAttempts = maxAttempts
	return o
}

func (o RetryOptions) DelayBetweenAttempts(delay time.Duration) RetryOptions {
	o.delayBetweenAttempts = delay
	return o
}

func (o RetryOptions) LogAttempts(logAttempts bool) RetryOptions {
	o.logAttempts = logAttempts
	return o
}
