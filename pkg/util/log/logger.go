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

package log

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

var Log = NewTextLogger()

type customFormatter struct {
}

func (c customFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	caller := entry.Caller
	callerFile := "unknown"
	callerLine := 0
	if caller != nil {
		callerFile = filepath.Base(caller.File)
		callerLine = caller.Line
	}

	// Every line is indented at least 4 spaces.
	b.WriteString("    ")
	if _, err := fmt.Fprintf(b, "%s:%d: ", callerFile, callerLine); err != nil {
		return b.Bytes(), err
	}
	lines := strings.Split(entry.Message, "\n")
	if l := len(lines); l > 1 && lines[l-1] == "" {
		lines = lines[:l-1]
	}
	for i, line := range lines {
		if i > 0 {
			// Second and subsequent lines are indented an additional 4 spaces.
			b.WriteString("\n        ")
		}
		b.WriteString(line)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

// NewTextLogger a StandardLogger with TextFormatter
func NewTextLogger() *logrus.Logger {
	var log = logrus.New()
	log.ReportCaller = true
	log.Formatter = &customFormatter{}
	return log
}
