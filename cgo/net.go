package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/Pie-Messaging/core/pie"
	"github.com/lucas-clemente/quic-go"
	"io"
	"net"
	"os"
	"runtime/cgo"
	"syscall"
	"time"
)

// #include <stdint.h>
// #include <sys/types.h>
import "C"

const (
	cgoTimeout = time.Second * 1
)

const (
	ENo int = iota
	EUnknown
	ETimedOut
	EClosed
	EMsgTooLong
	ECanceled
)

//export X509KeyPair
func X509KeyPair(certPEM []byte, keyPEM []byte, certDERResult []byte) C.uintptr_t {
	cert, err := pie.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return 0
	}
	copy(certDERResult, cert.Certificate[0])
	return C.uintptr_t(cgo.NewHandle(cert))
}

//export GenerateKeyPair
func GenerateKeyPair(certResult []byte, keyResult []byte, certDERResult []byte) C.uintptr_t {
	cert, certPEM, keyPEM, err := pie.GenerateKeyPair()
	if err != nil {
		return 0
	}
	copy(certResult, certPEM)
	copy(keyResult, keyPEM)
	copy(certDERResult, cert.Certificate[0])
	return C.uintptr_t(cgo.NewHandle(cert))
}

//export ListenNet
func ListenNet(listenAddr string, certPtr C.uintptr_t) (C.uintptr_t, int) {
	cert := cgo.Handle(certPtr).Value().(*tls.Certificate)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		NextProtos:   []string{pie.UserTLSProto},
	}
	for {
		server, err := pie.ListenNet(listenAddr, tlsConfig, nil)
		if err != nil {
			if errors.Is(err, syscall.EADDRINUSE) {
				listenAddr = ":0"
				continue
			}
			return 0, 0
		}
		port := server.Listener.Addr().(*net.UDPAddr).Port
		return C.uintptr_t(cgo.NewHandle(server)), port
	}
}

//export AcceptSession
func AcceptSession(ctxPtr C.uintptr_t, serverPtr C.uintptr_t, addrResult []byte) (C.uintptr_t, int, int) {
	ctx, cancel := context.WithTimeout(getContext(ctxPtr), cgoTimeout)
	defer cancel()
	server := cgo.Handle(serverPtr).Value().(*pie.Server)
	session, err := server.AcceptSession(ctx, true)
	if err != nil {
		errType := getErrType(err)
		if errType != ENo && errType != ETimedOut {
			pie.Logger.Println("Failed to accept session:", err)
		}
		return 0, 0, errType
	}
	addr := session.Session.RemoteAddr().String()
	copy(addrResult, addr)
	return C.uintptr_t(cgo.NewHandle(session)), len(addr), ENo
}

//export VerifyClientCert
func VerifyClientCert(serverPtr C.uintptr_t, clientCertDER []byte, serverCertSign []byte) bool {
	server := cgo.Handle(serverPtr).Value().(*pie.Server)
	if err := server.VerifyClientCert(clientCertDER, serverCertSign); err != nil {
		return false
	}
	return true
}

//export ConnectServer
func ConnectServer(ctxPtr C.uintptr_t, clientID []byte, clientCertPtr C.uintptr_t, serverAddr string, serverCertDER []byte) (C.uintptr_t, int) {
	ctx := getContext(ctxPtr)
	tlsConfig := &tls.Config{
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if !bytes.Equal(rawCerts[0], serverCertDER) {
				return errors.New("server certificate mismatch")
			}
			return nil
		},
		NextProtos:         []string{pie.UserTLSProto},
		InsecureSkipVerify: true,
	}
	session, err := pie.Connect(ctx, tlsConfig, serverAddr)
	if err != nil {
		return 0, getErrType(err)
	}
	cert := cgo.Handle(clientCertPtr).Value().(*tls.Certificate)
	err = session.SendCert(cert, clientID)
	pie.Logger.Println("Finished sending cert:", err)
	if err != nil {
		return 0, getErrType(err)
	}
	return C.uintptr_t(cgo.NewHandle(session)), ENo
}

//export ConnectTracker
func ConnectTracker(ctxPtr C.uintptr_t, addr string, idResult []byte) (C.uintptr_t, int) {
	ctx := getContext(ctxPtr)
	tlsConfig := &tls.Config{
		NextProtos:         []string{pie.UserTLSProto},
		InsecureSkipVerify: true,
	}
	session, err := pie.Connect(ctx, tlsConfig, addr)
	if err != nil {
		return 0, getErrType(err)
	}
	copy(idResult, session.GetPeerIDByCertHash())
	return C.uintptr_t(cgo.NewHandle(session)), ENo
}

//export SessionAcceptStream
func SessionAcceptStream(ctxPtr C.uintptr_t, sessionPtr C.uintptr_t, recvBuf []byte) (C.uintptr_t, int64, int) {
	ctx, cancel := context.WithTimeout(getContext(ctxPtr), cgoTimeout)
	defer cancel()
	session := cgo.Handle(sessionPtr).Value().(*pie.Session)
	stream, err := session.AcceptStream(ctx, recvBuf, true)
	if err != nil {
		errType := getErrType(err)
		if errType != ENo && errType != ETimedOut && errType != ECanceled {
			pie.Logger.Println("Failed to accept stream:", err)
		}
		return 0, -1, errType
	}
	return C.uintptr_t(cgo.NewHandle(stream)), int64(stream.Stream.StreamID()), ENo
}

//export SessionOpenStream
func SessionOpenStream(sessionPtr C.uintptr_t, recvBuf []byte) (C.uintptr_t, int64, int) {
	session := cgo.Handle(sessionPtr).Value().(*pie.Session)
	stream, err := session.OpenStream(recvBuf)
	if err != nil {
		return 0, -1, getErrType(err)
	}
	return C.uintptr_t(cgo.NewHandle(stream)), int64(stream.Stream.StreamID()), ENo
}

//export StreamRecvData
func StreamRecvData(streamPtr C.uintptr_t) (int, int, int) {
	_, start, end, err := cgo.Handle(streamPtr).Value().(*pie.Stream).RecvData(time.Now().Add(cgoTimeout), true)
	if err != nil {
		return -1, -1, getErrType(err)
	}
	return start, end, ENo
}

//export StreamSendData
func StreamSendData(streamPtr C.uintptr_t, data []byte, timeout int64) int {
	if err := cgo.Handle(streamPtr).Value().(*pie.Stream).SendData(data, getDeadline(timeout)); err != nil {
		return getErrType(err)
	}
	return ENo
}

//export CloseStream
func CloseStream(streamPtr C.uintptr_t) {
	handle := cgo.Handle(streamPtr)
	stream := handle.Value().(*pie.Stream)
	stream.Close()
	handle.Delete()
}

//export CloseSession
func CloseSession(sessionPtr C.uintptr_t, err uint64) {
	handle := cgo.Handle(sessionPtr)
	session := handle.Value().(*pie.Session)
	session.Close(err)
	handle.Delete()
}

//export CloseServer
func CloseServer(serverPtr C.uintptr_t) {
	handle := cgo.Handle(serverPtr)
	server := handle.Value().(*pie.Server)
	server.Close()
	handle.Delete()
}

//export DeleteCert
func DeleteCert(certPtr C.uintptr_t) {
	cgo.Handle(certPtr).Delete()
}

func getDeadline(timeout int64) time.Time {
	if timeout == 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(timeout) * time.Millisecond)
}

func getErrType(err error) int {
	errType := EUnknown
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		errType = ETimedOut
	} else if errors.Is(err, io.EOF) || errors.Is(err, &quic.IdleTimeoutError{}) {
		errType = EClosed
	} else if errors.Is(err, pie.ErrMsgTooLong) {
		errType = EMsgTooLong
	} else if errors.Is(err, context.Canceled) {
		errType = ECanceled
	}
	return errType
}
