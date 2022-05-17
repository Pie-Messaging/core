package main

import (
	"context"
	"runtime/cgo"
)

// #include <stdint.h>
// #include <sys/types.h>
import "C"

type Context struct {
	context.Context
	context.CancelFunc
}

//export NewContext
func NewContext() C.uintptr_t {
	ctx, cancelFunc := context.WithCancel(context.Background())
	return C.uintptr_t(cgo.NewHandle(Context{ctx, cancelFunc}))
}

//export CancelContext
func CancelContext(ctxPtr C.uintptr_t) {
	handle := cgo.Handle(ctxPtr)
	ctx := handle.Value().(Context)
	ctx.CancelFunc()
	handle.Delete()
}

func getContext(ctxPtr C.uintptr_t) context.Context {
	if ctxPtr == 0 {
		return context.Background()
	}
	return cgo.Handle(ctxPtr).Value().(Context).Context
}
