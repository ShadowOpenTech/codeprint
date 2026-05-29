package main

import "testing"

func TestGreet(t *testing.T) {
    if Greet("x") == "" { t.Fail() }
}
