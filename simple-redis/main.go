package main

import (
	"fmt"
	"io"

	"net"
	"os"
)

const addr = "0.0.0.0:6379"

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
