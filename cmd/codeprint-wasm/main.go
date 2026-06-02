//go:build js && wasm

// Command codeprint-wasm exposes codeprint.ScanFiles to JavaScript as a global
// fingerprint(inputJSON) function. inputJSON is {"files":{"<path>":"<content>"},
// "noFiles":bool}; the return is the canonical fingerprint JSON (string).
package main

import (
	"context"
	"encoding/json"
	"syscall/js"

	"github.com/ShadowOpenTech/codeprint/pkg/codeprint"
)

// fingerprint returns the fingerprint JSON string; throws a JS Error on
// failure. The thrown error (a real JS Error, surfaced by wasm_exec via panic)
// is caught by the JS worker's try/catch and is distinguishable from a
// successful JSON return.
func fingerprint(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		throw("missing input")
	}
	var in struct {
		Files   map[string]string `json:"files"`
		NoFiles bool              `json:"noFiles"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
		throw("bad input: " + err.Error())
	}
	files := make(map[string][]byte, len(in.Files))
	for p, c := range in.Files {
		files[p] = []byte(c)
	}
	var opts []codeprint.Option
	if in.NoFiles {
		opts = append(opts, codeprint.WithoutFileRecords())
	}
	fp, err := codeprint.ScanFiles(context.Background(), files, opts...)
	if err != nil {
		throw(err.Error())
	}
	out := codeprint.NewOutput(fp, "", "wasm", "")
	b, _ := json.Marshal(out)
	return string(b)
}

// throw raises a JS Error, which wasm_exec surfaces as a thrown JS exception
// for the caller's try/catch to handle.
func throw(msg string) {
	panic(js.Global().Get("Error").New(msg))
}

func main() {
	js.Global().Set("fingerprint", js.FuncOf(fingerprint))
	select {} // keep the module alive
}
