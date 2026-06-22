package ffmpeg

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// KeyInfo holds the material needed to encrypt HLS segments with AES-128 and to
// produce the ffmpeg key info file.
type KeyInfo struct {
	Key     []byte // 16-byte AES-128 key
	KeyURI  string // URI players use to fetch the key (written into the playlist)
	KeyPath string // path to the key file on disk (read by ffmpeg)
}

// GenerateKey returns a cryptographically random 16-byte AES-128 key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("encrypt: generate key: %w", err)
	}
	return key, nil
}

// KeyInfoContent renders the three-line ffmpeg key info file body:
//
//	<key URI>
//	<path to key file>
//	[optional IV in hex]
//
// ffmpeg reads line 1 as the URI to embed in the playlist, line 2 as the local
// key file to read, and the optional line 3 as the IV.
func KeyInfoContent(keyURI, keyPath, ivHex string) string {
	content := keyURI + "\n" + keyPath + "\n"
	if ivHex != "" {
		content += ivHex + "\n"
	}
	return content
}

// WriteKeyMaterial writes the key bytes and a key info file under dir, returning
// the populated KeyInfo. keyURI is the URI players will use to fetch the key.
func WriteKeyMaterial(dir, keyURI string, key []byte) (KeyInfo, error) {
	keyPath := filepath.Join(dir, "enc.key")
	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return KeyInfo{}, fmt.Errorf("encrypt: write key: %w", err)
	}
	keyInfoPath := filepath.Join(dir, "enc.keyinfo")
	if err := os.WriteFile(keyInfoPath, []byte(KeyInfoContent(keyURI, keyPath, "")), 0o600); err != nil {
		return KeyInfo{}, fmt.Errorf("encrypt: write key info: %w", err)
	}
	return KeyInfo{Key: key, KeyURI: keyURI, KeyPath: keyInfoPath}, nil
}

// KeyHex returns the key encoded as hex, useful for storing alongside metadata.
func KeyHex(key []byte) string {
	return hex.EncodeToString(key)
}
