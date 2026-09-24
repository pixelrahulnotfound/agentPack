// Package output provides agent-parseable CLI output: human text by
// default, structured JSON with --json. Errors always carry a machine
// readable code plus a hint.
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

type ErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Envelope struct {
	Ok    bool     `json:"ok"`
	Data  any      `json:"data,omitempty"`
	Error *ErrBody `json:"error,omitempty"`
}

var JSONMode = false

// Success prints data human-readably or as {"ok":true,"data":...}.
func Success(data any, human string) {
	if JSONMode {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(Envelope{Ok: true, Data: data})
		return
	}
	if human != "" {
		fmt.Println(human)
	} else {
		fmt.Printf("%v\n", data)
	}
}

// Failure prints a structured error to stderr and is meant to precede os.Exit(1).
func Failure(code, msg, hint string) {
	if JSONMode {
		enc := json.NewEncoder(os.Stderr)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(Envelope{Ok: false, Error: &ErrBody{Code: code, Message: msg, Hint: hint}})
		return
	}
	if hint != "" {
		fmt.Fprintf(os.Stderr, "error [%s]: %s\n  hint: %s\n", code, msg, hint)
		return
	}
	fmt.Fprintf(os.Stderr, "error [%s]: %s\n", code, msg)
}
