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

import (
	"io"
)

/* PostgreSQL Protocol Version/Code constants */
const (
	ProtocolVersion int32 = 196608
	SSLRequestCode  int32 = 80877103

	/* SSL Responses */
	SSLAllowed    byte = 'S'
	SSLNotAllowed byte = 'N'
)

/* PostgreSQL Message Type constants. */
const (
	AuthenticationMessageType  byte = 'R'
	ErrorMessageType           byte = 'E'
	EmptyQueryMessageType      byte = 'I'
	DescribeMessageType        byte = 'D'
	RowDescriptionMessageType  byte = 'T'
	DataRowMessageType         byte = 'D'
	QueryMessageType           byte = 'Q'
	CommandCompleteMessageType byte = 'C'
	TerminateMessageType       byte = 'X'
	NoticeMessageType          byte = 'N'
	PasswordMessageType        byte = 'p'
	ReadyForQueryMessageType   byte = 'Z'
)

/* PostgreSQL Authentication Method constants. */
const (
	AuthenticationOk          int32 = 0
	AuthenticationKerberosV5  int32 = 2
	AuthenticationClearText   int32 = 3
	AuthenticationMD5         int32 = 5
	AuthenticationSCM         int32 = 6
	AuthenticationGSS         int32 = 7
	AuthenticationGSSContinue int32 = 8
	AuthenticationSSPI        int32 = 9
)

func ReadMessageType(r io.Reader) (byte, error) {
	buf := []byte{0}
	_, err := r.Read(buf)
	return buf[0], err
}
