package ffmpeg

import (
	"strings"
	"testing"
)

func TestBuildMasterPlaylistHeader(t *testing.T) {
	m := BuildMasterPlaylist(DefaultLadder)
	if !strings.HasPrefix(m, "#EXTM3U\n") {
		t.Fatal("master playlist must start with #EXTM3U")
	}
	if !strings.Contains(m, "#EXT-X-VERSION:3") {
		t.Error("missing version tag")
	}
}

func TestBuildMasterPlaylistVariants(t *testing.T) {
	rends := []Rendition{
		{Name: "720p", Height: 720, Bandwidth: 3000000},
		{Name: "360p", Height: 360, Bandwidth: 900000},
	}
	m := BuildMasterPlaylist(rends)

	streamInfs := strings.Count(m, "#EXT-X-STREAM-INF")
	if streamInfs != 2 {
		t.Fatalf("expected 2 variant entries, got %d:\n%s", streamInfs, m)
	}
	if !strings.Contains(m, "BANDWIDTH=3000000,RESOLUTION=1280x720") {
		t.Errorf("missing/incorrect 720p stream-inf:\n%s", m)
	}
	if !strings.Contains(m, "BANDWIDTH=900000,RESOLUTION=640x360") {
		t.Errorf("missing/incorrect 360p stream-inf:\n%s", m)
	}
	if !strings.Contains(m, "\n720p.m3u8\n") || !strings.Contains(m, "\n360p.m3u8\n") {
		t.Errorf("variant playlist references missing:\n%s", m)
	}
}

func TestBuildMasterPlaylistOrderPreserved(t *testing.T) {
	rends := []Rendition{
		{Name: "1080p", Height: 1080, Bandwidth: 5500000},
		{Name: "480p", Height: 480, Bandwidth: 1600000},
	}
	m := BuildMasterPlaylist(rends)
	if strings.Index(m, "1080p.m3u8") > strings.Index(m, "480p.m3u8") {
		t.Error("rendition order should be preserved (1080p before 480p)")
	}
}
