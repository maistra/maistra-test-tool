// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
