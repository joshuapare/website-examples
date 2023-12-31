package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StoreItem struct {
	Value  string
	Expiry time.Time
	mutex  sync.RWMutex
}

// PerformPong response pack with "PONG", or optionally a passed in argument
func PerformPong(args []string, p *PersistenceEngine) string {
	if len(args) > 0 {
		return stringMsg(args[0])
	}
	return stringMsg("PONG")
}

// PerformEcho responds back with the passed in argument
func PerformEcho(args []string, p *PersistenceEngine) string {
	if len(args) == 0 {
		return errorMsg("no value provided to 'ECHO'")
	}
	return stringMsg(args[0])
}

// PerformSet stores a value with an expiry in the database
func PerformSet(args []string, p *PersistenceEngine) string {
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

	if p != nil {
		p.Log(&PersistenceLog{
			Command:   "SET",
			Arguments: args,
		})
	}
	return stringMsg("OK")
}

// PerformGet retrieves a value from the database, if it exists and is not expired. If it is expired, it will be deleted
func PerformGet(args []string, p *PersistenceEngine) string {
	if len(args) == 0 {
		return errorMsg("no value provided to 'GET'")
	}

	item := store[args[0]]

	// Check if it's null
	if item == nil {
		return nilBulkStringMsg()
	}

	// Allow simutaneous read, lock out write
	item.mutex.RLock()
	defer item.mutex.RUnlock()

	// Check the expiry
	now := time.Now()
	if item.Expiry.Before(now) {
		store[args[0]] = nil
		return nilBulkStringMsg()
	}

	return stringMsg(store[args[0]].Value)
}

// PerformDel deletes a value from the database, if it exists
func PerformDel(args []string, p *PersistenceEngine) string {
	if len(args) == 0 {
		return errorMsg("no value provided to 'DEL'")
	}

	item := store[args[0]]

	// Check if it's null
	if item == nil {
		return nilBulkStringMsg()
	}

	// Lock out read and write until write is done
	item.mutex.Lock()
	defer item.mutex.Unlock()

	// Delete the item
	delete(store, args[0])

	if p != nil {
		p.Log(&PersistenceLog{
			Command:   "DEL",
			Arguments: args,
		})
	}

	return stringMsg("OK")
}
