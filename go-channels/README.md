# Channels Logging Example

## Overview

The blog post for this example can be found [here](https://www.joshuapare.com/posts/using-go-channels).

This example serves as a short demonstration of how to use Go channels to implement a simple logging system, using a super inneficient implementation to simulate large wait times.

## Requirements

- Go 1.20 or higher

## Running the Server

To run the server, navigate to the `./go-channels` directory (if your are not already in it) and execute the following command:

```bash
go run main.go
```

This will start the server, which listens on port 9876 by default.

## Usage

To interact with the server, you can send use netcat:

```bash
echo 'some log message that you want to test with' | nc localhost 9876
```
