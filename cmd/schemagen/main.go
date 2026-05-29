// Command schemagen writes the canonical JSON schema to a file (or stdout) and
// can verify the committed schema is up to date (--check).
//
// Usage:
//
//	go run ./cmd/schemagen -out schema/codeprint-v1.schema.json
//	go run ./cmd/schemagen -check -out schema/codeprint-v1.schema.json
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/ShadowOpenTech/codeprint/internal/schemagen"
)

func main() {
	out := flag.String("out", "", "output file (default stdout)")
	check := flag.String("check", "", "verify the file at this path matches generated schema; exit 1 on drift")
	flag.Parse()

	data, err := schemagen.Generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "schemagen:", err)
		os.Exit(1)
	}

	if *check != "" {
		existing, err := os.ReadFile(*check)
		if err != nil {
			fmt.Fprintln(os.Stderr, "schemagen: read", *check, err)
			os.Exit(1)
		}
		if !bytes.Equal(existing, data) {
			fmt.Fprintln(os.Stderr, "schemagen: DRIFT — committed schema differs from generated. Run: make schema")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "schemagen: schema up to date")
		return
	}

	if *out == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			fmt.Fprintln(os.Stderr, "schemagen:", err)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*out, data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "schemagen: write", *out, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "schemagen: wrote", *out)
}
