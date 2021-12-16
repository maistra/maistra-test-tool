// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"github.com/sirupsen/logrus"
)

var Log = NewTextLogger()

// Event contains log id and log message.
type Event struct {
	id      int
	message string
}

// StandardLogger a wrapper of logrus.Logger
type StandardLogger struct {
	*logrus.Logger
}

// NewTextLogger a StandardLogger with TextFormatter
func NewTextLogger() *StandardLogger {
	var baseLogger = logrus.New()
	var standardLogger = &StandardLogger{baseLogger}
	standardLogger.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}

	return standardLogger
}

// NewJSONLogger a StandardLogger with JOSNFormatter
func NewJSONLogger() *StandardLogger {
	var baseLogger = logrus.New()
	var standardLogger = &StandardLogger{baseLogger}
	standardLogger.Formatter = &logrus.JSONFormatter{}

	return standardLogger
}
