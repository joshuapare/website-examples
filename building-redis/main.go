package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"net"
	"os"
)

type StoreItem struct {
	Value  string
	Expiry time.Time
}

var (
	store = make(map[string]*StoreItem)
	addr  = "0.0.0.0:6379"
)

func main() {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to bind to %s\n", addr)
		os.Exit(1)
	}

	fmt.Printf("Starting Redis server at %s\n", addr)
	// Listen for inputs and respond
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go func(conn net.Conn) {
			defer conn.Close()
			buf := make([]byte, 2014) // store out stuff somewhere

			for {
				len, err := conn.Read(buf)
				if err != nil {
					if err != io.EOF {
						fmt.Printf("Error reading: %#v\n", err)
					}
					break
				}

				// Parse the command out
				command := buf[:len]
				response := Parse(command)

				// Write the response back to the connection
				_, responseErr := conn.Write([]byte(response + "\r\n"))
				if responseErr != nil {
					fmt.Printf("Error reading: %#v\n", err)
					break
				}
			}
		}(conn)
	}
}

type Parser struct {
	// The number of expected arguments we should see while parsing
	NumberOfExpectedArguments int
	// The character length of the next argument to parse
	LengthOfNextArgument int
	// The amount of arguments parsed
	NumberOfArgumentsParsed int
}

func Parse(command []byte) string {
	var cmd string
	args := []string{}

	parser := &Parser{
		NumberOfExpectedArguments: 0,
		LengthOfNextArgument:      0,
		NumberOfArgumentsParsed:   0,
	}

	position := 0
	for position < len(command) {
		switch command[position] {
		case '*':
			result, err := getIntArg(position+1, command)
			if err != nil {
				return err.Error()
			}
			// Enforce the number of expected arguments, and preallocate array without the first
			// argument, which will be the first command
			parser.NumberOfExpectedArguments = result.Result
			args = make([]string, parser.NumberOfExpectedArguments-1)

			position += result.PositionsParsed
		case '$':
			result, err := getIntArg(position+1, command)
			if err != nil {
				return err.Error()
			}
			// Enforce the length of the next argument
			parser.LengthOfNextArgument = result.Result

			position += result.PositionsParsed
		case '\r':
			position += 1
		case '\n':
			position += 1
		default:
			// Make sure we haven't reached here improperly through invalid argument syntax
			if parser.NumberOfExpectedArguments == 0 || parser.LengthOfNextArgument == 0 || parser.NumberOfArgumentsParsed >= parser.NumberOfExpectedArguments {
				return stringMsg("Invalid syntax")
			}

			// We've gotten past the checks - parse it!
			parsedItem := string(command[position : parser.LengthOfNextArgument+position])

			if parser.NumberOfArgumentsParsed > 0 {
				// it's an arg, add to args array
				args[parser.NumberOfArgumentsParsed-1] = parsedItem
			} else {
				// The first 'arg' we parse is the primary command
				cmd = parsedItem
			}
			// Move our position forward, as we have parsed the argument
			position += parser.LengthOfNextArgument

			// Prepare our parser for the next parsing sequence
			parser.LengthOfNextArgument = 0
			parser.NumberOfArgumentsParsed += 1
		}
	}

	return ParseCommand(cmd, args)
}

// ParseCommand parses the incoming
func ParseCommand(command string, args []string) string {
	cmd := strings.ToLower(command)
	switch cmd {
	case "ping":
		return PerformPong(args)
	case "echo":
		return PerformEcho(args)
	case "set":
		return PerformSet(args)
	case "get":
		return PerformGet(args)
	default:
		return errorMsg(fmt.Sprintf("unknown command '%s'", cmd))
	}
}

// PerformPong response pack with "PONG", or optionally a passed in argument
func PerformPong(args []string) string {
	if len(args) > 0 {
		return stringMsg(args[0])
	}
	return stringMsg("PONG")
}

// PerformEcho responds back with the passed in argument
func PerformEcho(args []string) string {
	if len(args) == 0 {
		return errorMsg("no value provided to 'ECHO'")
	}
	return stringMsg(args[0])
}

// PerformSet stores a value with an expiry in the database
func PerformSet(args []string) string {
	if len(args) < 2 {
		return errorMsg("invalid syntax provided to 'SET'")
	}
	key := &args[0]
	val := &args[1]
	var exp time.Time

	if len(args) > 2 {
		// We have options to parse. Let's parse them!
		position := 2
		for position < len(args) {
			switch strings.ToLower(args[position]) {
			case "px":
				// Set expiry in milliseconds
				if len(args) < position+1 {
					return errorMsg("no time provided to 'PX'")
				}
				expMillis, err := strconv.Atoi(string(args[position+1]))
				if err != nil {
					return errorMsg("invalid format provided to 'PX'")
				}
				exp = time.Now().Add(time.Millisecond * time.Duration(expMillis))

				position += 2
			case "ex":
				// Set expiry in seconds
				if len(args) < position+1 {
					return errorMsg("no time provided to 'EX'")
				}
				expSeconds, err := strconv.Atoi(string(args[position+1]))
				if err != nil {
					return errorMsg("invalid format provided to 'EX'")
				}
				exp = time.Now().Add(time.Second * time.Duration(expSeconds))

				position += 2
			default:
				return errorMsg(fmt.Sprintf("invalid argument '%s'", args[position]))
			}
		}
	}

	// Default to an hour
	if exp == (time.Time{}) {
		exp = time.Now().Add(time.Hour)
	}

	store[*key] = &StoreItem{
		Value:  *val,
		Expiry: exp,
	}
	return stringMsg("OK")
}

func PerformGet(args []string) string {
	if len(args) == 0 {
		return errorMsg("no value provided to 'GET'")
	}

	// Perform expiry invalidations on GETs
	item := store[args[0]]

	// Check if it's null
	if item == nil {
		return nilBulkStringMsg()
	}
	now := time.Now()
	// Check the expiry
	if item.Expiry.Before(now) {
		store[args[0]] = nil
		return nilBulkStringMsg()
	}

	return stringMsg(store[args[0]].Value)
}

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
