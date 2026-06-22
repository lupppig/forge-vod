package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	k1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if len(k1) != 16 {
		t.Fatalf("expected 16-byte key, got %d", len(k1))
	}
	k2, _ := GenerateKey()
	if string(k1) == string(k2) {
		t.Fatal("two generated keys are identical; randomness broken")
	}
}

func TestKeyInfoContent(t *testing.T) {
	got := KeyInfoContent("https://cdn/key", "/tmp/enc.key", "")
	want := "https://cdn/key\n/tmp/enc.key\n"
	if got != want {
		t.Fatalf("KeyInfoContent = %q, want %q", got, want)
	}
	withIV := KeyInfoContent("uri", "path", "deadbeef")
	if !strings.HasSuffix(withIV, "deadbeef\n") {
		t.Fatalf("expected IV appended, got %q", withIV)
	}
}

func TestWriteKeyMaterial(t *testing.T) {
	dir := t.TempDir()
	key, _ := GenerateKey()

	ki, err := WriteKeyMaterial(dir, "https://cdn/key", key)
	if err != nil {
		t.Fatalf("WriteKeyMaterial: %v", err)
	}
	if ki.KeyURI != "https://cdn/key" {
		t.Errorf("unexpected KeyURI %q", ki.KeyURI)
	}

	onDisk, err := os.ReadFile(filepath.Join(dir, "enc.key"))
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	if string(onDisk) != string(key) {
		t.Error("key file content does not match generated key")
	}
	info, err := os.ReadFile(ki.KeyPath)
	if err != nil {
		t.Fatalf("read key info: %v", err)
	}
	if !strings.Contains(string(info), "https://cdn/key") {
		t.Error("key info missing key URI")
	}
}

func TestRenditionArgsWithEncryption(t *testing.T) {
	r := Rendition{Name: "480p", Height: 480, VideoKbps: 1400, AudioKbps: 128}
	args := RenditionArgsWithOptions("in.mp4", "/out", r, EncodeOptions{KeyInfoFile: "/out/enc.keyinfo"})
	if !containsPair(args, "-hls_key_info_file", "/out/enc.keyinfo") {
		t.Error("expected hls_key_info_file when encryption enabled")
	}
	// Without encryption the flag must be absent.
	plain := RenditionArgs("in.mp4", "/out", r, 6)
	for _, a := range plain {
		if a == "-hls_key_info_file" {
			t.Error("unencrypted args should not contain key info flag")
		}
	}
}

func TestEncryptedRenditionIntegration(t *testing.T) {
	dir := t.TempDir()
	src := makeTestVideo(t, dir)
	outDir := filepath.Join(dir, "hls")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	key, _ := GenerateKey()
	ki, err := WriteKeyMaterial(outDir, "enc.key", key)
	if err != nil {
		t.Fatalf("WriteKeyMaterial: %v", err)
	}

	r := Rendition{Name: "240p", Height: 240, VideoKbps: 400, AudioKbps: 64}
	err = EncodeRenditionWithOptions(context.Background(), NewExecRunner(), "ffmpeg", src, outDir, r,
		EncodeOptions{SegmentDuration: 2, KeyInfoFile: ki.KeyPath})
	if err != nil {
		t.Fatalf("EncodeRenditionWithOptions: %v", err)
	}

	playlist, err := os.ReadFile(filepath.Join(outDir, "240p.m3u8"))
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if !strings.Contains(string(playlist), "#EXT-X-KEY") {
		t.Fatalf("playlist not encrypted, missing EXT-X-KEY:\n%s", playlist)
	}
	if !strings.Contains(string(playlist), "METHOD=AES-128") {
		t.Fatalf("expected AES-128 method in playlist:\n%s", playlist)
	}
}
