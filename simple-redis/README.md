# Simple Redis

## Overview

The Simple Redis implementation in Go serves as a minimalist, educational example of a Redis-like key-value store. This server is compatible with `redis-cli` and supports the following commands:

- `PING`
- `ECHO`
- `SET` (with `PX` and `EX` options for setting expiry)
- `GET`

## Requirements

- Go 1.20 or higher

## Running the Server

To run the server, navigate to the `./simple-redis` directory and execute the following command:

```bash
go run main.go
```

This will start the Simple Redis server, which listens on port 6379 by default.

## Usage

To interact with the Simple Redis server, you can use the official `redis-cli`.

For example, after starting the server, open a new terminal and run:

```bash
redis-cli
```

Inside the Redis CLI, you can then execute the following commands:

### PING

```bash
PING hello
```

### ECHO

```bash
ECHO "Hello, World!"
```

### SET

Set a value with no expiry:

```bash
SET key value
```

Set a value with an expiry of 1000 milliseconds:

```bash
SET key value PX 1000
```

Set a value with an expiry of 10 seconds:

```bash
SET key value EX 10
```

### GET

```bash
GET key
```

## Code Structure

- `main.go`: Entry point of the application, initializes the server.
- `commands.go`: Contains the implementations for the Redis commands.
- `parser.go`: Contains utility functions for parsing Redis commands and arguments.
- `utils.go`: Contains utility functions for the server.

## Contributing

If you'd like to contribute to this project, feel free to open an issue or create a pull request.