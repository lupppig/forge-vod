package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid url: %w", err)
	}

	client := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}

	return client, nil
}

const TranscodeStream = "transcode:jobs"

type Queue struct {
	client *redis.Client
}

func NewQueue(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// EnqueueTranscode appends a transcode job for the given video to the
// transcode stream. The worker consumes this stream to produce HLS renditions.
func (q *Queue) EnqueueTranscode(ctx context.Context, videoID, objectKey string) error {
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: TranscodeStream,
		Values: map[string]any{
			"video_id":   videoID,
			"object_key": objectKey,
		},
	}).Err()
}

// TranscodeGroup is the consumer group the worker reads the transcode stream
// with, giving at-least-once delivery and pending-entry tracking.
const TranscodeGroup = "transcoders"

// Job is a single transcode job read from the stream.
type Job struct {
	ID        string // stream entry id, used to ack
	VideoID   string
	ObjectKey string
}

// Consumer reads transcode jobs from the stream via a consumer group.
type Consumer struct {
	client   *redis.Client
	consumer string
}

// NewConsumer returns a Consumer identified by name within the group.
func NewConsumer(client *redis.Client, name string) *Consumer {
	return &Consumer{client: client, consumer: name}
}

// EnsureGroup creates the consumer group if it does not already exist, starting
// from the beginning of the stream. It is safe to call on every startup.
func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, TranscodeStream, TranscodeGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("redis: create group: %w", err)
	}
	return nil
}

// Read blocks up to block for the next new job in the group. It returns
// (nil, nil) when the block elapses with no new entries.
func (c *Consumer) Read(ctx context.Context, block time.Duration) (*Job, error) {
	res, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    TranscodeGroup,
		Consumer: c.consumer,
		Streams:  []string{TranscodeStream, ">"},
		Count:    1,
		Block:    block,
	}).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis: read group: %w", err)
	}
	for _, stream := range res {
		for _, msg := range stream.Messages {
			return &Job{
				ID:        msg.ID,
				VideoID:   asString(msg.Values["video_id"]),
				ObjectKey: asString(msg.Values["object_key"]),
			}, nil
		}
	}
	return nil, nil
}

// Ack acknowledges a processed job so it leaves the pending list.
func (c *Consumer) Ack(ctx context.Context, id string) error {
	return c.client.XAck(ctx, TranscodeStream, TranscodeGroup, id).Err()
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
