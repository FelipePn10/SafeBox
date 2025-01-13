package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// EncryptStream encrypts the contents of a stream in chunks
func EncryptStream(input io.Reader, output io.Writer, key []byte) error {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return errors.New("invalid key size: must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	// Write nonce to the output
	if _, err := output.Write(nonce); err != nil {
		return err
	}

	buffer := make([]byte, 4096) // Process file in 4KB chunks
	for {
		n, err := input.Read(buffer)
		if n > 0 {
			chunk := gcm.Seal(nil, nonce, buffer[:n], nil)
			if _, err := output.Write(chunk); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// DecryptStream decrypts the contents of a stream in chunks
func DecryptStream(input io.Reader, output io.Writer, key []byte) error {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return errors.New("invalid key size: must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonceSize := gcm.NonceSize()
	nonce := make([]byte, nonceSize)

	// Read nonce from the input
	if _, err := io.ReadFull(input, nonce); err != nil {
		return err
	}

	buffer := make([]byte, 4096+gcm.Overhead()) // Account for GCM overhead
	for {
		n, err := input.Read(buffer)
		if n > 0 {
			chunk, err := gcm.Open(nil, nonce, buffer[:n], nil)
			if err != nil {
				return err
			}
			if _, err := output.Write(chunk); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// GenerateEncryptionKey generates a 256-bit encryption key and returns it as a byte slice
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) // 256 bits for AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, errors.New("failed to generate encryption key")
	}
	return key, nil
}

// KeyToHex converts a key to a hexadecimal string
func KeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}

// KeyFromHex converts a hexadecimal string back to a key
func KeyFromHex(hexKey string) ([]byte, error) {
	return hex.DecodeString(hexKey)
}
