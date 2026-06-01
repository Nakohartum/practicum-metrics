package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
)

type Envelope struct {
	Key        string `json:"key"`
	Nonce      string `json:"nonce"`
	CipherText string `json:"ciphertext"`
}

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to read")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)

	if err != nil {
		return nil, err
	}

	publicKey, ok := key.(*rsa.PublicKey)

	if !ok {
		return nil, errors.New("not RSA key")
	}

	return publicKey, nil
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)

	if block == nil {
		return nil, errors.New("failed to read block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)

	if err == nil {
		return privateKey, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPrivateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not RSA private key")
	}

	return rsaPrivateKey, nil
}

func Encrypt(data []byte, key *rsa.PublicKey) ([]byte, error) {
	aesKey := make([]byte, 32)

	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	cipherText := gcm.Seal(nil, nonce, data, nil)

	encryptedKey, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		key,
		aesKey,
		nil,
	)

	if err != nil {
		return nil, err
	}

	envelope := Envelope{
		Key:        base64.StdEncoding.EncodeToString(encryptedKey),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		CipherText: base64.StdEncoding.EncodeToString(cipherText),
	}

	return json.Marshal(envelope)
}

func Decrypt(data []byte, key *rsa.PrivateKey) ([]byte, error) {
	var envelope Envelope

	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(envelope.Key)

	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(envelope.CipherText)
	if err != nil {
		return nil, err
	}

	aesKey, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		key,
		encryptedKey,
		nil,
	)

	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plainData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plainData, nil
}
