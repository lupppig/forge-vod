package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lupppig/forge-vod/internal/repository/video"
)

func TestToAPIVideoBuildsAbsoluteURLs(t *testing.T) {
	h := &VideoHandler{publicStreamURL: "https://cdn.example.com/streams"}
	id := uuid.New()
	v := &video.Video{
		ID:            id,
		Title:         "ready clip",
		Status:        video.StatusReady,
		ObjectKey:     id.String() + ".mp4",
		MasterKey:     id.String() + "/master.m3u8",
		ThumbnailKey:  id.String() + "/thumbnail.jpg",
		StoryboardKey: id.String() + "/storyboard.vtt",
		Duration:      42.5,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	out := h.toAPIVideo(v)

	if out.MasterUrl == nil || *out.MasterUrl != "https://cdn.example.com/streams/"+id.String()+"/master.m3u8" {
		t.Errorf("unexpected master url: %v", out.MasterUrl)
	}
	if out.ThumbnailUrl == nil || *out.ThumbnailUrl != "https://cdn.example.com/streams/"+id.String()+"/thumbnail.jpg" {
		t.Errorf("unexpected thumbnail url: %v", out.ThumbnailUrl)
	}
	if out.StoryboardUrl == nil || *out.StoryboardUrl != "https://cdn.example.com/streams/"+id.String()+"/storyboard.vtt" {
		t.Errorf("unexpected storyboard url: %v", out.StoryboardUrl)
	}
	if out.Duration == nil || *out.Duration != 42.5 {
		t.Errorf("unexpected duration: %v", out.Duration)
	}
}

func TestToAPIVideoOmitsURLsWhenPending(t *testing.T) {
	h := &VideoHandler{publicStreamURL: "https://cdn.example.com/streams"}
	v := &video.Video{
		ID:        uuid.New(),
		Title:     "pending clip",
		Status:    video.StatusPending,
		ObjectKey: "raw.mp4",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	out := h.toAPIVideo(v)

	if out.MasterUrl != nil || out.ThumbnailUrl != nil || out.StoryboardUrl != nil {
		t.Errorf("expected no playback URLs before processing, got %+v", out)
	}
	if out.Duration != nil {
		t.Errorf("expected nil duration before processing, got %v", *out.Duration)
	}
}
