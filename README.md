# beterr

[![Go Reference](https://pkg.go.dev/badge/github.com/StevenM2002/beterr.svg)](https://pkg.go.dev/github.com/StevenM2002/beterr)
[![Go Report Card](https://goreportcard.com/badge/github.com/StevenM2002/beterr)](https://goreportcard.com/report/github.com/StevenM2002/beterr)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight Go package for structured error handling and debugging with enhanced error formatting, function call context, and argument inspection.

## Features

-  **Structured Error Formatting**: Wrap errors with function context and arguments for better debugging
-  **Automatic Call Stack Information**: Capture function names using runtime reflection
-  **Argument Inspection**: Include function arguments in error output
-  **JSON Serialization**: Convert complex data structures to readable JSON format
-  **Error Chaining**: Support for nested error structures with full context preservation
-  **Zero Dependencies**: Uses only Go standard library
-  **Lightweight**: Minimal performance overhead

## Installation

```bash
go get github.com/StevenM2002/beterr
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/StevenM2002/beterr"
)

func processUser(userID int, name string) error {
    debug := beterr.Debug{A: []any{userID, name}}
    
    if userID < 0 {
        return debug.E(fmt.Errorf("invalid user ID"), "failed to process user")
    }
    
    return nil
}

func main() {
    err := processUser(-1, "John")
    if err != nil {
        fmt.Println(err)
        // Output: {"fn_name":"main.processUser","args":[-1,"John"],"msg":"failed to process user","inner":"invalid user ID"}
    }
}
```

## Usage Examples

### Basic Error Wrapping

```go
func validateInput(data string) error {
    debug := beterr.Debug{A: []any{data}}
    
    if len(data) == 0 {
        return debug.E(fmt.Errorf("empty input"), "validation failed")
    }
    
    return nil
}
```

### With Context and Complex Types

```go
import "context"

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func handleRequest(ctx context.Context, user *User) error {
    debug := beterr.Debug{A: []any{ctx, user}}
    
    err := validateUser(user)
    if err != nil {
        return debug.E(err, "request processing failed")
    }
    
    return nil
}
```

### Struct Serialization Utility

```go
type Config struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

config := Config{Host: "localhost", Port: 8080}
jsonStr := beterr.StructString(config)
fmt.Println(jsonStr) // {"host":"localhost","port":8080}
```

## API Reference

### Types

#### Debug

```go
type Debug struct {
    A []any // Arguments to include in debug output
}
```

The `Debug` struct is the main type for creating structured error contexts. The `A` field holds arguments that will be serialized and included in the error output.

### Methods

#### E(err error, msg ...string) error

Formats an error with debugging context including:
- Function name (automatically captured)
- Arguments from the `A` field
- Custom message
- Original error (supports chaining)

**Parameters:**
- `err`: The original error to wrap
- `msg`: Optional message parts that will be joined with spaces

**Returns:** A new error with structured debugging information

### Functions

#### StructString(v any) string

Converts any value to a JSON string representation. If JSON marshaling fails, it falls back to Go's default string formatting.

**Parameters:**
- `v`: Any value to serialize

**Returns:** JSON string representation or fallback string format

## Error Output Format

The package produces structured JSON error output with the following fields:

```json
{
  "fn_name": "main.processUser",
  "args": ["-1", "\"John\""],
  "msg": "failed to process user", 
  "inner": "invalid user ID"
}
```

- `fn_name`: Fully qualified function name where the error occurred
- `args`: JSON-serialized arguments passed to the Debug struct
- `msg`: Custom error message
- `inner`: The original error or nested error structure

## Error Chaining

The package supports full error chaining, preserving the complete context chain:

```go
func level1() error {
    debug := beterr.Debug{A: []any{"level1-arg"}}
    return debug.E(level2(), "level1 failed")
}

func level2() error {
    debug := beterr.Debug{A: []any{"level2-arg"}}
    return debug.E(fmt.Errorf("original error"), "level2 failed")
}
```

This creates a nested structure showing the complete error path with context at each level.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by the need for better error debugging in Go applications
- Built with Go's excellent standard library for runtime introspection