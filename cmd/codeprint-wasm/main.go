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

func fingerprint(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errJSON("missing input")
	}
	var in struct {
		Files   map[string]string `json:"files"`
		NoFiles bool              `json:"noFiles"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
		return errJSON("bad input: " + err.Error())
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
		return errJSON(err.Error())
	}
	out := codeprint.NewOutput(fp, "", "wasm", "")
	b, _ := json.Marshal(out)
	return string(b)
}

func errJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

func main() {
	js.Global().Set("fingerprint", js.FuncOf(fingerprint))
	select {} // keep the module alive
}
