package bukanir

import (
	"context"
	"errors"
	"io"

	youtube "github.com/kkdai/youtube/v2"
)

// Trailer resolves a YouTube video id to a stream URL.
func (c *Client) Trailer(ctx context.Context, videoID string) (string, error) {
	yt, video, format, err := c.trailerFormat(ctx, videoID)
	if err != nil {
		return "", err
	}
	return yt.GetStreamURLContext(ctx, video, format)
}

// TrailerStream resolves a trailer and returns its video stream.
func (c *Client) TrailerStream(ctx context.Context, videoID string) (io.ReadCloser, int64, error) {
	yt, video, format, err := c.trailerFormat(ctx, videoID)
	if err != nil {
		return nil, 0, err
	}
	return yt.GetStreamContext(ctx, video, format)
}

func (c *Client) trailerFormat(ctx context.Context, videoID string) (youtube.Client, *youtube.Video, *youtube.Format, error) {
	hc, err := c.tr.client()
	if err != nil {
		return youtube.Client{}, nil, nil, err
	}
	youtube.DefaultClient = youtube.AndroidClient
	yt := youtube.Client{HTTPClient: hc}
	video, err := yt.GetVideoContext(ctx, videoID)
	if err != nil {
		return yt, nil, nil, err
	}
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return yt, nil, nil, errors.New("bukanir: no playable trailer format")
	}
	format := &formats[0]
	if q := formats.Quality("360p"); len(q) > 0 {
		format = &q[0]
	}
	return yt, video, format, nil
}
