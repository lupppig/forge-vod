package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"

	"github.com/lupppig/forge-vod/internal/ffmpeg"
)

// ObjectStore is the subset of storage the pipeline needs. Defining it here lets
// tests substitute a fake without a live MinIO.
type ObjectStore interface {
	DownloadRaw(ctx context.Context, objectKey, dst string) error
	UploadHLSDir(ctx context.Context, localDir, prefix string) error
}

// Encoder is the subset of ffmpeg operations the pipeline drives. The real
// implementation wraps the ffmpeg package; tests use a recording fake.
type Encoder interface {
	Probe(ctx context.Context, input string) (ffmpeg.MediaInfo, error)
	WriteKeyMaterial(dir, keyURI string, key []byte) (ffmpeg.KeyInfo, error)
	GenerateKey() ([]byte, error)
	EncodeRendition(ctx context.Context, input, outDir string, r ffmpeg.Rendition, opts ffmpeg.EncodeOptions) error
	GenerateThumbnail(ctx context.Context, input, out string, at float64, width int) error
	GenerateStoryboard(ctx context.Context, input, out string, s ffmpeg.StoryboardSpec) error
	WriteFile(name string, data []byte) error
}

// Pipeline runs the full per-video processing flow: probe, transcode each
// rendition (AES-128 encrypted), thumbnail, storyboard preview, then the master
// manifest, finally uploading the output tree.
type Pipeline struct {
	Store   ObjectStore
	Encoder Encoder
	Log     *slog.Logger
	KeyURI  string // key URI embedded in encrypted playlists
}

// Result summarizes what the pipeline produced for a video.
type Result struct {
	Renditions   []ffmpeg.Rendition
	MasterPath   string
	ThumbnailKey string
}

// Process runs every stage for one video. workDir is a scratch directory the
// caller is responsible for cleaning up; outPrefix is the key prefix the HLS
// output is uploaded under in the HLS bucket.
func (p *Pipeline) Process(ctx context.Context, videoID, objectKey, workDir, outPrefix string) (*Result, error) {
	log := p.Log.With(slog.String("video_id", videoID))

	srcPath := filepath.Join(workDir, "source"+path.Ext(objectKey))
	outDir := filepath.Join(workDir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	log.Info("stage: download source", slog.String("object_key", objectKey))
	if err := p.Store.DownloadRaw(ctx, objectKey, srcPath); err != nil {
		return nil, err
	}

	log.Info("stage: probe")
	info, err := p.Encoder.Probe(ctx, srcPath)
	if err != nil {
		return nil, err
	}

	ladder := ffmpeg.LadderFor(info.Height, ffmpeg.DefaultLadder)
	log.Info("planned ladder", slog.Int("source_height", info.Height), slog.Int("renditions", len(ladder)))

	log.Info("stage: hls encryption key")
	key, err := p.Encoder.GenerateKey()
	if err != nil {
		return nil, err
	}
	keyInfo, err := p.Encoder.WriteKeyMaterial(outDir, p.KeyURI, key)
	if err != nil {
		return nil, err
	}

	log.Info("stage: transcode/encode renditions")
	for _, r := range ladder {
		if err := p.Encoder.EncodeRendition(ctx, srcPath, outDir, r, ffmpeg.EncodeOptions{
			SegmentDuration: 6,
			KeyInfoFile:     keyInfo.KeyPath,
		}); err != nil {
			return nil, err
		}
		log.Info("encoded rendition", slog.String("name", r.Name))
	}

	log.Info("stage: thumbnail")
	thumbPath := filepath.Join(outDir, "thumbnail.jpg")
	if err := p.Encoder.GenerateThumbnail(ctx, srcPath, thumbPath, ffmpeg.ThumbnailTimestamp(info.Duration), 640); err != nil {
		return nil, err
	}

	log.Info("stage: preview storyboard")
	spec := ffmpeg.DefaultStoryboard()
	spritePath := filepath.Join(outDir, "storyboard.jpg")
	if err := p.Encoder.GenerateStoryboard(ctx, srcPath, spritePath, spec); err != nil {
		return nil, err
	}
	vtt := ffmpeg.BuildVTT(ffmpeg.SpriteName(spritePath), info.Duration, spec)
	if err := p.Encoder.WriteFile(filepath.Join(outDir, "storyboard.vtt"), []byte(vtt)); err != nil {
		return nil, err
	}

	log.Info("stage: master manifest")
	master := ffmpeg.BuildMasterPlaylist(ladder)
	masterPath := filepath.Join(outDir, "master.m3u8")
	if err := p.Encoder.WriteFile(masterPath, []byte(master)); err != nil {
		return nil, err
	}

	log.Info("stage: upload output", slog.String("prefix", outPrefix))
	if err := p.Store.UploadHLSDir(ctx, outDir, outPrefix); err != nil {
		return nil, err
	}

	return &Result{
		Renditions:   ladder,
		MasterPath:   fmt.Sprintf("%s/master.m3u8", outPrefix),
		ThumbnailKey: fmt.Sprintf("%s/thumbnail.jpg", outPrefix),
	}, nil
}
