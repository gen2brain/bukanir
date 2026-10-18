package bukanir

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	youtube "github.com/kkdai/youtube/v2"
)

// trailerMaxHeight is the tallest trailer video picked.
const trailerMaxHeight = 720

// trailerClient is the YouTube player client used for trailers; Android is refused the 720p streams of most videos.
var trailerClient = youtube.ClientInfo{
	Name:        "VISIONOS",
	Version:     "1.02",
	UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15",
	DeviceModel: "RealityDevice17,1",
}

// trailerClientOnce sets the YouTube client once: every kkdai request, stream chunks included, reads the global.
var trailerClientOnce sync.Once

// TrailerMedia is one opened stream of a trailer.
type TrailerMedia struct {
	Body     io.ReadCloser
	Size     int64
	MimeType string
}

// Trailer resolves a YouTube video id to its video stream URL and, for a video-only format, its audio stream URL.
func (c *Client) Trailer(ctx context.Context, videoID string) (videoURL, audioURL string, err error) {
	yt, video, choice, err := c.trailerFormats(ctx, videoID)
	if err != nil {
		return "", "", err
	}
	vf, af := choice.muxed, (*youtube.Format)(nil)
	if v, a, ok := c.openTrailerPair(ctx, yt, video, choice); ok {
		v.Body.Close()
		a.Body.Close()
		vf, af = choice.video, choice.audio
	}
	if videoURL, err = yt.GetStreamURLContext(ctx, video, vf); err != nil {
		return "", "", err
	}
	if af != nil {
		if audioURL, err = yt.GetStreamURLContext(ctx, video, af); err != nil {
			return "", "", err
		}
	}
	return videoURL, audioURL, nil
}

// TrailerStreams opens a trailer's video stream and, for a video-only format, its audio stream.
func (c *Client) TrailerStreams(ctx context.Context, videoID string) (TrailerMedia, *TrailerMedia, error) {
	yt, video, choice, err := c.trailerFormats(ctx, videoID)
	if err != nil {
		return TrailerMedia{}, nil, err
	}
	if v, a, ok := c.openTrailerPair(ctx, yt, video, choice); ok {
		return v, &a, nil
	}
	v, err := openTrailerMedia(ctx, yt, video, choice.muxed)
	return v, nil, err
}

// trailerChoice holds the separate video and audio formats of a trailer and its muxed fallback.
type trailerChoice struct {
	video, audio, muxed *youtube.Format
}

// openTrailerPair opens the separate video and audio streams; YouTube refuses them for some videos.
func (c *Client) openTrailerPair(ctx context.Context, yt youtube.Client, video *youtube.Video, choice trailerChoice) (TrailerMedia, TrailerMedia, bool) {
	if choice.video == nil || choice.audio == nil {
		return TrailerMedia{}, TrailerMedia{}, false
	}
	v, err := openTrailerMedia(ctx, yt, video, choice.video)
	if err != nil {
		logf("TRAILER: %dp video unavailable, using muxed: %v", choice.video.Height, err)
		return TrailerMedia{}, TrailerMedia{}, false
	}
	a, err := openTrailerMedia(ctx, yt, video, choice.audio)
	if err != nil {
		v.Body.Close()
		logf("TRAILER: audio unavailable, using muxed: %v", err)
		return TrailerMedia{}, TrailerMedia{}, false
	}
	return v, a, true
}

// openTrailerMedia opens one format and reads ahead, since a refused stream fails on its first read.
func openTrailerMedia(ctx context.Context, yt youtube.Client, video *youtube.Video, f *youtube.Format) (TrailerMedia, error) {
	if f == nil {
		return TrailerMedia{}, errors.New("bukanir: no playable trailer format")
	}
	body, size, err := yt.GetStreamContext(ctx, video, f)
	if err != nil {
		return TrailerMedia{}, err
	}
	br := bufio.NewReaderSize(body, 32<<10)
	if _, err := br.Peek(1); err != nil {
		body.Close()
		return TrailerMedia{}, err
	}
	return TrailerMedia{Body: peekedBody{br, body}, Size: size, MimeType: trailerMime(f)}, nil
}

// peekedBody reads through the buffer that already holds the peeked bytes.
type peekedBody struct {
	*bufio.Reader
	io.Closer
}

func (c *Client) trailerFormats(ctx context.Context, videoID string) (youtube.Client, *youtube.Video, trailerChoice, error) {
	hc, err := c.tr.client()
	if err != nil {
		return youtube.Client{}, nil, trailerChoice{}, err
	}
	trailerClientOnce.Do(func() { youtube.DefaultClient = trailerClient })
	yt := youtube.Client{HTTPClient: hc}
	video, err := yt.GetVideoContext(ctx, videoID)
	if err != nil {
		return yt, nil, trailerChoice{}, err
	}
	choice := pickTrailerFormats(video.Formats)
	if choice.muxed == nil && (choice.video == nil || choice.audio == nil) {
		return yt, nil, trailerChoice{}, errors.New("bukanir: no playable trailer format")
	}
	return yt, video, choice, nil
}

// pickTrailerFormats picks H.264 video up to trailerMaxHeight with the best AAC track, and the tallest muxed format.
func pickTrailerFormats(formats youtube.FormatList) trailerChoice {
	var choice trailerChoice
	for i := range formats {
		f := &formats[i]
		switch {
		case f.AudioChannels == 0 && strings.HasPrefix(f.MimeType, "video/mp4") && strings.Contains(f.MimeType, "avc1") && f.Height <= trailerMaxHeight:
			if choice.video == nil || f.Height > choice.video.Height || (f.Height == choice.video.Height && f.Bitrate > choice.video.Bitrate) {
				choice.video = f
			}
		case f.AudioChannels > 0 && strings.HasPrefix(f.MimeType, "audio/mp4"):
			if choice.audio == nil || f.Bitrate > choice.audio.Bitrate {
				choice.audio = f
			}
		case f.AudioChannels > 0 && strings.HasPrefix(f.MimeType, "video/") && f.Height <= trailerMaxHeight:
			if choice.muxed == nil || f.Height > choice.muxed.Height {
				choice.muxed = f
			}
		}
	}
	return choice
}

func trailerMime(f *youtube.Format) string {
	mime, _, _ := strings.Cut(f.MimeType, ";")
	return mime
}
