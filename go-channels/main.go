package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

const addr = "0.0.0.0:9876"

func main() {
	listen, err := net.Listen("tcp", addr)
	defer listen.Close()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Started server on %s\n", addr)

	theLogger := &Logger{
		Channel: make(chan []byte),
	}

	go theLogger.startLogger()

	for {
		conn, err := listen.Accept()
		start := time.Now()

		if err != nil {
			fmt.Println()
		}
		buffer := make([]byte, 2048)

		// Read into the buffer
		num, err := conn.Read(buffer)

		// // Log it to a file
		// stupidlyLogToFile(buffer[:num])

		theLogger.log(buffer[:num])

		// Echo back to the connection
		resp := fmt.Sprintf("Time to execute: %v\n", time.Since(start))
		conn.Write([]byte(resp))
		conn.Close()
	}
}

type Logger struct {
	// Logging channel to recieve logs from
	Channel chan []byte
}

// startLogger starts the logger on a channel
func (l *Logger) startLogger() {
	// Start the logger listening on the channel
	fmt.Println("Starting logging channel")
	for {
		toLog := <-l.Channel
		stupidlyLogToFile(toLog)
	}
}

// log logs a message
func (l *Logger) log(info []byte) {
	// Send the info through the channel
	fmt.Println("Logging info")
	l.Channel <- info
}

// stupidlyLogToFile stupidly logs to a file by logging out each character one at a time
func stupidlyLogToFile(log []byte) {
	// Stupidly log out each character one by one to the file
	for i := 0; i < len(log); i++ {
		// Create the file if it doesn't exist
		if _, err := os.Stat("log.txt"); errors.Is(err, os.ErrNotExist) {
			os.Create("log.txt")
		}

		// Open it for writing
		f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Println("failed to open file")
		}
		defer f.Close()

		// Write out the single byte
		if _, err := f.Write([]byte{log[i]}); err != nil {
			fmt.Println("failed to to log file")
		}
	}
}
