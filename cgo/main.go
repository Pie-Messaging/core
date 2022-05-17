package main

import (
	"github.com/Pie-Messaging/core/pie"
	"golang.org/x/crypto/sha3"
)

import "C"

//export SetLogOutput
func SetLogOutput(path string) {
	pie.SetLogOutput(path)
}

//export HashBytes
func HashBytes(data []byte, result []byte) {
	sha3.ShakeSum256(result, data)
}

func main() {}
