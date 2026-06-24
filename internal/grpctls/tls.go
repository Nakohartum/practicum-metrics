package grpctls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc/credentials"
)

const (
	certDir  = "certs"
	certFile = "grpc-cert.pem"
	keyFile  = "grpc-key.pem"
)

func ClientCredentials() (credentials.TransportCredentials, error) {
	certPEM, _, err := ensureCertificateFiles()
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("append gRPC root certificate")
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: "localhost",
		RootCAs:    pool,
	}), nil
}

func ServerCredentials() (credentials.TransportCredentials, error) {
	certPEM, keyPEM, err := ensureCertificateFiles()
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load gRPC certificate: %w", err)
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}), nil
}

func ensureCertificateFiles() ([]byte, []byte, error) {
	certPath := filepath.Join(certDir, certFile)
	keyPath := filepath.Join(certDir, keyFile)

	certPEM, certErr := os.ReadFile(certPath)
	keyPEM, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		return certPEM, keyPEM, nil
	}
	if !os.IsNotExist(certErr) && certErr != nil {
		return nil, nil, fmt.Errorf("read gRPC certificate: %w", certErr)
	}
	if !os.IsNotExist(keyErr) && keyErr != nil {
		return nil, nil, fmt.Errorf("read gRPC private key: %w", keyErr)
	}

	certPEM, keyPEM, err := generateCertificate()
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create gRPC certificate directory: %w", err)
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, nil, fmt.Errorf("write gRPC certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, nil, fmt.Errorf("write gRPC private key: %w", err)
	}

	return certPEM, keyPEM, nil
}

func generateCertificate() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate gRPC private key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().AddDate(10, 0, 0),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		DNSNames: []string{"localhost"},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create gRPC certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal gRPC private key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}
