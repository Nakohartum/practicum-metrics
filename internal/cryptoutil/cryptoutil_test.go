package cryptoutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "public.pem")
	err = os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}), 0600)
	require.NoError(t, err)

	publicKey, err := LoadPublicKey(path)

	require.NoError(t, err)
	assert.Equal(t, privateKey.PublicKey.N, publicKey.N)
	assert.Equal(t, privateKey.PublicKey.E, publicKey.E)
}

func TestLoadPrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "private.pem")
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	err = os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}), 0600)
	require.NoError(t, err)

	got, err := LoadPrivateKey(path)

	require.NoError(t, err)
	assert.Equal(t, privateKey.N, got.N)
	assert.Equal(t, privateKey.E, got.E)
	assert.Equal(t, privateKey.D, got.D)
}

func TestEncryptDecrypt(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	message := []byte(`{"id":"hits","type":"counter","delta":3}`)

	encrypted, err := Encrypt(message, &privateKey.PublicKey)
	require.NoError(t, err)
	assert.NotEqual(t, message, encrypted)

	decrypted, err := Decrypt(encrypted, privateKey)
	require.NoError(t, err)
	assert.Equal(t, message, decrypted)
}

func TestDecryptRejectsInvalidEnvelope(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	_, err = Decrypt([]byte("not-json"), privateKey)

	require.Error(t, err)
}
