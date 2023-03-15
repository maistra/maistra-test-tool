package retry

import "time"

var defaultOptions = RetryOptions{
	maxAttempts:          30,
	delayBetweenAttempts: 1 * time.Second,
}

type RetryOptions struct {
	maxAttempts          int
	delayBetweenAttempts time.Duration
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
