package proxy

import (
	"bytes"
	"encoding/hex"
	"io"

	"github.com/brunopadz/mammoth/protocol"
)

const typeField = "type"

type pgArg struct {
	Fmt   string
	Value string
}

// See description of FunctionCall or Bind below for the type of data that
// this function parses.
func handleArgs(m *protocol.Reader) ([]pgArg, error) {
	argFmtCount16, err := m.ReadInt16()
	if err != nil {
		return nil, err
	}
	argFmtCount := int(argFmtCount16)

	lastFmtCode := 0
	argFmtCodes := make([]int, argFmtCount)
	for i := 0; i < argFmtCount; i++ {
		fmtCode16, err := m.ReadInt16()
		if err != nil {
			return nil, err
		}
		lastFmtCode = int(fmtCode16)
		argFmtCodes[i] = lastFmtCode
	}

	argCnt16, err := m.ReadInt16()
	if err != nil {
		return nil, err
	}
	argCnt := int(argCnt16)

	args := make([]pgArg, argCnt)

	for i := 0; i < argCnt; i++ {
		argLen, err := m.ReadInt32()
		if err != nil {
			return args[:i], err
		}
		if argLen == -1 {
			args[i] = pgArg{Fmt: "null", Value: ""}
			continue
		}

		argBuf := make([]byte, argLen)

		_, err = io.ReadFull(m, argBuf)
		if err != nil {
			return args[:i], err
		}

		var argFmtCode int
		if len(argFmtCodes) <= i {
			argFmtCode = int(lastFmtCode)
		} else {
			argFmtCode = argFmtCodes[i]
		}

		var argFmt string
		var argValue string

		switch argFmtCode {
		case 0:
			argFmt = "text"
			// we'll try our best, but we don't detect client encoding
			argValue = string(argBuf)
		default:
			argFmt = "binary"
			argValue = hex.EncodeToString(argBuf)
		}

		args[i] = pgArg{Fmt: argFmt, Value: argValue}
	}
	return args, nil
}

/*
CopyData (F & B)
Byte1('d')
Identifies the message as COPY data.

Int32
Length of message contents in bytes, including self.

Byten
Data that forms part of a COPY data stream. Messages sent from the backend will always correspond to single data rows, but messages sent by frontends might divide the data stream arbitrarily.
*/
func handleCopyData(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "CopyData"

	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, m)

	fields["value"] = hex.EncodeToString(buf.Bytes())
	if err != nil {
		return err
	}

	return m.Finalize()
}

/*
CopyDone (F & B)
Byte1('c')
Identifies the message as a COPY-complete indicator.

Int32(4)
Length of message contents in bytes, including self.
*/
func handleCopyDone(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "CopyDone"
	return m.Finalize()
}

/*
CopyFail (F)
Byte1('f')
Identifies the message as a COPY-failure indicator.

Int32
Length of message contents in bytes, including self.

String
An error message to report as the cause of failure.
*/
func handleCopyFail(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "CopyFail"

	msg, err := m.ReadString()
	fields["errorMessage"] = msg
	if err != nil {
		return err
	}

	return m.Finalize()
}

/*
Bind (F)
Byte1('B')
Identifies the message as a Bind command.

Int32
Length of message contents in bytes, including self.

String
The name of the destination portal (an empty string selects the unnamed portal).

String
The name of the source prepared statement (an empty string selects the unnamed prepared statement).

Int16
The number of parameter format codes that follow (denoted C below). This can be zero to indicate that there are no parameters or that the parameters all use the default format (text); or one, in which case the specified format code is applied to all parameters; or it can equal the actual number of parameters.

Int16[C]
The parameter format codes. Each must presently be zero (text) or one (binary).

Int16
The number of parameter values that follow (possibly zero). This must match the number of parameters needed by the query.

Next, the following pair of fields appear for each parameter:

Int32
The length of the parameter value, in bytes (this count does not include itself). Can be zero. As a special case, -1 indicates a NULL parameter value. No value bytes follow in the NULL case.

Byten
The value of the parameter, in the format indicated by the associated format code. n is the above length.

After the last parameter, the following fields appear:

Int16
The number of result-column format codes that follow (denoted R below). This can be zero to indicate that there are no result columns or that the result columns should all use the default format (text); or one, in which case the specified format code is applied to all result columns (if any); or it can equal the actual number of result columns of the query.

Int16[R]
The result-column format codes. Each must presently be zero (text) or one (binary).
*/
func handleBind(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Bind"

	name, err := m.ReadString()
	fields["portal"] = name
	if err != nil {
		return err
	}

	args, err := handleArgs(m)
	fields["args"] = args
	if err != nil {
		return err
	}

	return m.Discard()
}

/*
Close (F)
Byte1('C')
Identifies the message as a Close command.

Int32
Length of message contents in bytes, including self.

Byte1
'S' to close a prepared statement; or 'P' to close a portal.

String
The name of the prepared statement or portal to close (an empty string selects the unnamed prepared statement or portal).
*/
func handleClose(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Close"

	target := "unknown"
	targetB, err := m.ReadByte()
	switch targetB {
	case 'S':
		target = "prepared"
	case 'P':
		target = "portal"
	}
	fields["target"] = target
	if err != nil {
		return err
	}

	name, err := m.ReadString()
	fields["name"] = name
	if err != nil {
		return err
	}

	return m.Finalize()
}

/*
Describe (F)
Byte1('D')
Identifies the message as a Describe command.

Int32
Length of message contents in bytes, including self.

Byte1
'S' to describe a prepared statement; or 'P' to describe a portal.

String
The name of the prepared statement or portal to describe (an empty string selects the unnamed prepared statement or portal).
*/
func handleDescribe(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Describe"

	target := "unknown"
	targetB, err := m.ReadByte()
	switch targetB {
	case 'S':
		target = "prepared"
	case 'P':
		target = "portal"
	}
	fields["target"] = target
	if err != nil {
		return err
	}

	name, err := m.ReadString()
	fields["name"] = name
	if err != nil {
		return err
	}

	return m.Finalize()
}

/*
Execute (F)
Byte1('E')
Identifies the message as an Execute command.

Int32
Length of message contents in bytes, including self.

String
The name of the portal to execute (an empty string selects the unnamed portal).

Int32
Maximum number of rows to return, if portal contains a query that returns rows (ignored otherwise). Zero denotes “no limit”.
*/
func handleExecute(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Execute"

	portal, err := m.ReadString()
	fields["portalName"] = portal

	if err != nil {
		return err
	}

	maxRows, err := m.ReadInt32()
	fields["maxRows"] = maxRows
	if err != nil {
		return err
	}

	return m.Finalize()
}

/*
Query (F)
Byte1('Q')
Identifies the message as a simple query.

Int32
Length of message contents in bytes, including self.

String
The query string itself.
*/
func handleSimpleQuery(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "SimpleQuery"

	s, err := m.ReadString()
	fields["query"] = s
	if err != nil {
		return err
	}

	return m.Finalize()
}

/*
Parse (F)
Byte1('P')
Identifies the message as a Parse command.

Int32
Length of message contents in bytes, including self.

String
The name of the destination prepared statement (an empty string selects the unnamed prepared statement).

String
The query string to be parsed.

Int16
The number of parameter data types specified (can be zero). Note that this is not an indication of the number of parameters that might appear in the query string, only the number that the frontend wants to prespecify types for.

Then, for each parameter, there is the following:

Int32
Specifies the object ID of the parameter data type. Placing a zero here is equivalent to leaving the type unspecified.
*/
func handleParse(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Parse"

	prepStmt, err := m.ReadString()
	fields["preparedStatement"] = prepStmt
	if err != nil {
		return err
	}

	query, err := m.ReadString()
	fields["query"] = query
	if err != nil {
		return err
	}

	return m.Discard()
}

/*
FunctionCall (F)
Byte1('F')
Identifies the message as a function call.

Int32
Length of message contents in bytes, including self.

Int32
Specifies the object ID of the function to call.

Int16
The number of argument format codes that follow (denoted C below). This can be zero to indicate that there are no arguments or that the arguments all use the default format (text); or one, in which case the specified format code is applied to all arguments; or it can equal the actual number of arguments.

Int16[C]
The argument format codes. Each must presently be zero (text) or one (binary).

Int16
Specifies the number of arguments being supplied to the function.

Next, the following pair of fields appear for each argument:

Int32
The length of the argument value, in bytes (this count does not include itself). Can be zero. As a special case, -1 indicates a NULL argument value. No value bytes follow in the NULL case.

Byten
The value of the argument, in the format indicated by the associated format code. n is the above length.

After the last argument, the following field appears:

Int16
The format code for the function result. Must presently be zero (text) or one (binary).
*/
func handleFunctionCall(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "FunctionCall"

	oid, err := m.ReadInt16()
	fields["funcOID"] = oid
	if err != nil {
		return err
	}

	args, err := handleArgs(m)
	fields["args"] = args
	if err != nil {
		return err
	}

	return m.Discard()
}

/*
Sync (F)
Byte1('S')
Identifies the message as a Sync command.

Int32(4)
Length of message contents in bytes, including self.
*/
func handleSync(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Sync"
	return m.Finalize()
}

/*
Terminate (F)
Byte1('X')
Identifies the message as a termination.

Int32(4)
Length of message contents in bytes, including self.
*/
func handleTerminate(m *protocol.Reader, fields map[string]interface{}) error {
	fields[typeField] = "Terminate"
	return m.Finalize()
}
