package pie

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
)

func X509KeyPair(certPEM, keyPEM []byte) (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		Logger.Println("Failed to load key pair:", err)
		return nil, err
	}
	return &cert, nil
}

func GenerateKeyPair() (*tls.Certificate, []byte, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		Logger.Println("Failed to generate key pair:", err)
		return nil, nil, nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey, privateKey)
	certPEM := pem.EncodeToMemory(
		&pem.Block{
			Type: "CERTIFICATE", Bytes: certDER,
		},
	)
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		Logger.Println("Failed to marshal private key:", err)
		return nil, nil, nil, err
	}
	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type: "PRIVATE KEY", Bytes: privateKeyBytes,
		},
	)
	certificate, err := tls.X509KeyPair(certPEM, privateKeyPEM)
	if err != nil {
		Logger.Println("Failed to load key pair:", err)
		return nil, nil, nil, err
	}
	return &certificate, certPEM, privateKeyPEM, nil
}
