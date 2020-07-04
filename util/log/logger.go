/*
Copyright 2017 Crunchy Data Solutions, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package log

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
)

var levels = []string{
	"debug",
	"info",
	"error",
	"fatal",
}

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetOutput(os.Stdout)
}

func Debug(msg string) {
	logrus.Debug(msg)
}

func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

func Info(msg string) {
	logrus.Info(msg)
}

func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func Error(msg string) {
	logrus.Error(msg)
}

func Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func Fatal(msg string) {
	logrus.Fatal(msg)
}

func Fatalf(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

func WithFields(f logrus.Fields) *logrus.Entry {
	return logrus.WithFields(f)
}

func SetLevel(level string) error {
	logrusLevel, err := logrus.ParseLevel(level)

	if err != nil {
		return err
	}

	logrus.SetLevel(logrusLevel)
	return nil
}

func SetFormat(format string) error {
	switch format {
	case "plain":
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		return fmt.Errorf("Unknown log format: %s", format)
	}
	return nil
}
