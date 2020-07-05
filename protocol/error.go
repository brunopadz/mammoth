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

package protocol

import "io"

/* PG Error Severity Levels */
const (
	// "error" errors
	ErrorSeverityError string = "ERROR"
	ErrorSeverityFatal string = "FATAL"
	ErrorSeverityPanic string = "PANIC"
	// "notice" errors
	ErrorSeverityWarning string = "WARNING"
	ErrorSeverityNotice  string = "NOTICE"
	ErrorSeverityDebug   string = "DEBUG"
	ErrorSeverityInfo    string = "INFO"
	ErrorSeverityLog     string = "LOG"
)

/* PG Error Message Field Identifiers */
const (
	ErrorFieldSeverity      byte = 'S'
	ErrorFieldCode          byte = 'C'
	ErrorFieldMessage       byte = 'M'
	ErrorFieldMessageDetail byte = 'D'
	ErrorFieldMessageHint   byte = 'H'
)

const (
	ErrorCodeInternalError         string = "XX000"
	ErrorCodeFeatureNotSupported   string = "0A000"
	ErrorCodeConnectionFailure     string = "08006"
	ErrorCodeClientUnableToConnect string = "08001"
	ErrorCodeServerRejected        string = "08004"
)

type Error struct {
	Severity string
	Code     string
	Message  string
	Detail   string
	Hint     string
}

func WriteError(w io.Writer, e Error) error {
	_, err := w.Write([]byte{'E'})
	if err != nil {
		return err
	}

	msg := NewBuffer()
	msg.WriteByte(ErrorFieldSeverity)
	if e.Severity == "" {
		msg.WriteString(ErrorSeverityError)
	} else {
		msg.WriteString(e.Severity)
	}
	msg.WriteByte(ErrorFieldCode)
	if e.Code == "" {
		msg.WriteString(ErrorCodeInternalError)
	} else {
		msg.WriteString(e.Code)
	}
	msg.WriteByte(ErrorFieldMessage)
	msg.WriteString(e.Message)
	if e.Detail != "" {
		msg.WriteByte(ErrorFieldMessageDetail)
		msg.WriteString(e.Detail)
	}
	if e.Hint != "" {
		msg.WriteByte(ErrorFieldMessageHint)
		msg.WriteString(e.Hint)
	}
	msg.WriteByte(0)
	return msg.WriteTo(w)
}
