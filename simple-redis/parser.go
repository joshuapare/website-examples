package main

import (
	"fmt"
	"strings"
)

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
