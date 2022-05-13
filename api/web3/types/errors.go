// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import "fmt"

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

type DataError interface {
	Error() string          // returns the message
	ErrorData() interface{} // returns the error data
}

// Error types defined below are the built-in JSON-RPC errors.

var (
	_ Error = new(MethodNotFoundError)
	_ Error = new(subscriptionNotFoundError)
	_ Error = new(ParseError)
	_ Error = new(InvalidRequestError)
	_ Error = new(invalidMessageError)
	_ Error = new(InvalidParamsError)
)

const defaultErrorCode = -32000

type MethodNotFoundError struct{ Method string }

func (e *MethodNotFoundError) ErrorCode() int { return -32601 }

func (e *MethodNotFoundError) Error() string {
	return fmt.Sprintf("the method %s does not exist/is not available", e.Method)
}

type subscriptionNotFoundError struct{ namespace, subscription string }

func (e *subscriptionNotFoundError) ErrorCode() int { return -32601 }

func (e *subscriptionNotFoundError) Error() string {
	return fmt.Sprintf("no %q subscription in %s namespace", e.subscription, e.namespace)
}

// Invalid JSON was received by the server.
type ParseError struct{ Message string }

func (e *ParseError) ErrorCode() int { return -32700 }

func (e *ParseError) Error() string { return e.Message }

// received message isn't a valid request
type InvalidRequestError struct{ Message string }

func (e *InvalidRequestError) ErrorCode() int { return -32600 }

func (e *InvalidRequestError) Error() string { return e.Message }

// received message is invalid
type invalidMessageError struct{ message string }

func (e *invalidMessageError) ErrorCode() int { return -32700 }

func (e *invalidMessageError) Error() string { return e.message }

// unable to decode supplied params, or an invalid number of parameters
type InvalidParamsError struct{ Message string }

func (e *InvalidParamsError) ErrorCode() int { return -32602 }

func (e *InvalidParamsError) Error() string { return e.Message }
