package pie

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"github.com/Pie-Messaging/core/pie/pb"
	"github.com/lucas-clemente/quic-go"
)

var (
	quicConfig = &quic.Config{KeepAlive: true}
)

type Session struct {
	Session quic.EarlySession
}

func Connect(ctx context.Context, tlsConfig *tls.Config, addrList ...string) (*Session, error) {
	// TODO: support multiple addresses
	for _, addr := range addrList {
		session, err := quic.DialAddrEarlyContext(ctx, addr, tlsConfig, quicConfig)
		if err != nil {
			Logger.Println("Failed to connect:", err)
			return nil, err
		}
		return &Session{Session: session}, nil
	}
	return nil, ErrNoAddr
}

func (s *Session) AcceptStream(ctx context.Context, recvBuf []byte, noLog ...bool) (*Stream, error) {
	stream, err := s.Session.AcceptStream(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			Log(noLog, "Failed to accept stream:", err)
		}
		return nil, err
	}
	return NewStream(stream, recvBuf), nil
}

func (s *Session) OpenStream(recvBuf ...[]byte) (*Stream, error) {
	stream, err := s.Session.OpenStream()
	if err != nil {
		Logger.Println("Failed to open stream:", err)
		return nil, err
	}
	return NewStream(stream, recvBuf...), nil
}

func (s *Session) SendCert(cert *tls.Certificate, id ...[]byte) error {
	stream, err := s.OpenStream()
	if err != nil {
		return err
	}
	Logger.Println("Sending cert: stream id:", stream.Stream.StreamID())
	serverCertDER := s.Session.ConnectionState().TLS.PeerCertificates[0].Raw
	serverCertHash := HashBytes(serverCertDER, ServerCertHashLen)
	sign, err := cert.PrivateKey.(crypto.Signer).Sign(rand.Reader, serverCertHash, crypto.Hash(0))
	if err != nil {
		Logger.Println("Failed to sign hash:", err)
		return err
	}
	if len(id) == 0 {
		id = append(id, nil)
	}
	return stream.SendMessage(&pb.NetMessage{Body: &pb.NetMessage_ClientCertReq{
		ClientCertReq: &pb.ClientCertReq{
			Id:             id[0],
			CertDer:        cert.Certificate[0],
			ServerCertSign: sign,
		},
	}})
}

func (s *Session) GetPeerIDByCertHash() []byte {
	return HashBytes(s.Session.ConnectionState().TLS.PeerCertificates[0].Raw, IDLen)
}

func (s *Session) Close(errCode uint64) {
	err := s.Session.CloseWithError(quic.ApplicationErrorCode(errCode), "")
	if err != nil {
		Logger.Println("Failed to close session:", err)
		return
	}
}
