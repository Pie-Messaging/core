package pie

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"github.com/lucas-clemente/quic-go"
)

type Server struct {
	Listener quic.EarlyListener
	CertHash []byte
}

func ListenNet(listenAddr string, tlsConfig *tls.Config, quicConfig_ *quic.Config) (*Server, error) {
	listener, err := quic.ListenAddrEarly(listenAddr, tlsConfig, quicConfig)
	if err != nil {
		Logger.Println("Failed to listen net:", err)
		return nil, err
	}
	server := &Server{Listener: listener, CertHash: HashBytes(tlsConfig.Certificates[0].Certificate[0], ServerCertHashLen)}
	return server, nil
}

func (s *Server) AcceptSession(ctx context.Context, noLog ...bool) (*Session, error) {
	sess, err := s.Listener.Accept(ctx)
	if err != nil {
		Log(noLog, "Failed to accept session:", err)
		return nil, err
	}
	session := &Session{Session: sess}
	return session, nil
}

func (s *Server) VerifyClientCert(clientCertDER []byte, serverCertSign []byte) error {
	cert, err := x509.ParseCertificate(clientCertDER)
	if err != nil {
		Logger.Println("Failed to parse certificate:", err)
		return err
	}
	if !ed25519.Verify(cert.PublicKey.(ed25519.PublicKey), s.CertHash, serverCertSign) {
		Logger.Println("Failed to verify server cert sign:", err)
		return err
	}
	return nil
}

func (s *Server) Close() {
	err := s.Listener.Close()
	if err != nil {
		Logger.Println("Failed to close server:", err)
	}
}
