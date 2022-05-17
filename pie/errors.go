package pie

import "C"
import (
	"errors"
)

var (
	ErrProtoEOF   = errors.New("protobuf bytes EOF")
	ErrMsgTooLong = errors.New("message too long")
	ErrEmptyMsg   = errors.New("empty message")
	ErrInvalidMsg = errors.New("invalid message")
)

var (
	ErrNoAddr = errors.New("no available address")
)

const (
	SessErrNoReason = iota
	SessErrNotFound
)
