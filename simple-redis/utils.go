package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// errorMsg returns a formatted string error message
func errorMsg(msg string) string {
	return fmt.Sprintf("-ERR %s", msg)
}

// stringMsg returns a formatted string message
func stringMsg(msg string) string {
	return "+" + msg
}

// nilBulkStringMsg returns a nil bulk string Msg
func nilBulkStringMsg() string {
	return "$-1"
}

type GetIntArgResult struct {
	// The result of the parsing operation
	Result int
	// How many indexes were parsed from the provided byte array
	PositionsParsed int
}

// getIntArg parses an integer argument from a string representation and returns
// the result and skipped amount of bytes in the array
func getIntArg(startPosition int, arr []byte) (*GetIntArgResult, error) {
	var err error
	result := &GetIntArgResult{
		Result:          0,
		PositionsParsed: 0,
	}

	// Get digits until termination characters
	notDone := true
	position := startPosition

	// Building out a string with concatenation creates a new string each time. String builder
	// is more efficient for incrementally building a string
	var stringVal strings.Builder
	for notDone {
		if arr[position] == '\r' && arr[position+1] == '\n' {
			// We hit the termination, consider it done
			notDone = false
		} else {
			stringVal.WriteByte(arr[position])
		}

		result.PositionsParsed += 1
		position += 1
	}

	if stringVal.Len() == 0 {
		return nil, errors.New("no value was detected")
	}

	resultInt, err := strconv.Atoi(stringVal.String())
	if err != nil {
		return nil, errors.New("failed to parse int")
	}

	result.Result = resultInt
	return result, nil
}
