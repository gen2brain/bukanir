package bukanir

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	youtube "github.com/kkdai/youtube/v2"
)

func TestTrailer(t *testing.T) {
	if testing.Short() {
		t.Skip("resolves a real YouTube stream; skipped under -short")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// "Me at the zoo" — the first, and very stable, YouTube video.
	url, _, err := c.Trailer(ctx, "jNQXAC9IVRw")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http") || !strings.Contains(url, "googlevideo.com") {
		t.Fatalf("unexpected trailer URL: %q", url)
	}
}

func TestTrailerBadID(t *testing.T) {
	if testing.Short() {
		t.Skip("contacts YouTube; skipped under -short")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, _, err := c.Trailer(ctx, "!!!not-a-video!!!"); err == nil {
		t.Fatal("expected error for invalid video id")
	}
}

func TestTrailerStream(t *testing.T) {
	if testing.Short() {
		t.Skip("streams a real YouTube video; skipped under -short")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	start := time.Now()
	s, err := c.Summary(ctx, 660120, CategoryMovies, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("summary in %v, trailer key %q", time.Since(start), s.TrailerKey)
	if s.TrailerKey == "" {
		t.Skip("no trailer key for the movie")
	}

	start = time.Now()
	video, audio, err := c.TrailerStreams(ctx, s.TrailerKey)
	if err != nil {
		t.Fatal(err)
	}
	defer video.Body.Close()
	t.Logf("streams resolved in %v, video %s %d bytes", time.Since(start), video.MimeType, video.Size)

	start = time.Now()
	buf := make([]byte, 256<<10)
	n, err := io.ReadFull(video.Body, buf)
	t.Logf("read %d video bytes in %v, err %v", n, time.Since(start), err)
	if n == 0 {
		t.Fatal("no data from the trailer video stream")
	}
	if audio != nil {
		defer audio.Body.Close()
		t.Logf("audio %s %d bytes", audio.MimeType, audio.Size)
		if n, _ := io.ReadFull(audio.Body, buf); n == 0 {
			t.Fatal("no data from the trailer audio stream")
		}
	}
}

func TestPickTrailerFormats(t *testing.T) {
	formats := youtube.FormatList{
		{ItagNo: 137, URL: "u", MimeType: `video/mp4; codecs="avc1.640028"`, Height: 1080, Bitrate: 4351220},
		{ItagNo: 247, URL: "u", MimeType: `video/webm; codecs="vp9"`, Height: 720, Bitrate: 1562786},
		{ItagNo: 136, URL: "u", MimeType: `video/mp4; codecs="avc1.4D401F"`, Height: 720, Bitrate: 1441705},
		{ItagNo: 398, URL: "u", MimeType: `video/mp4; codecs="av01.0.05M.08"`, Height: 720, Bitrate: 1111264},
		{ItagNo: 18, URL: "u", MimeType: `video/mp4; codecs="avc1.42001E, mp4a.40.2"`, Height: 360, Bitrate: 665745, AudioChannels: 2},
		{ItagNo: 140, URL: "u", MimeType: `audio/mp4; codecs="mp4a.40.2"`, Bitrate: 130470, AudioChannels: 2},
		{ItagNo: 251, URL: "u", MimeType: `audio/webm; codecs="opus"`, Bitrate: 126232, AudioChannels: 2},
		{ItagNo: 139, URL: "u", MimeType: `audio/mp4; codecs="mp4a.40.5"`, Bitrate: 49886, AudioChannels: 2},
	}
	choice := pickTrailerFormats(formats)
	if choice.video == nil || choice.video.ItagNo != 136 || choice.audio == nil || choice.audio.ItagNo != 140 {
		t.Fatalf("picked video %v audio %v, want 136 and 140", choice.video, choice.audio)
	}
	if choice.muxed == nil || choice.muxed.ItagNo != 18 {
		t.Errorf("muxed fallback = %v, want 18", choice.muxed)
	}
	if trailerMime(choice.video) != "video/mp4" || trailerMime(choice.audio) != "audio/mp4" {
		t.Errorf("mime = %q, %q", trailerMime(choice.video), trailerMime(choice.audio))
	}

	choice = pickTrailerFormats(formats[:5])
	if choice.audio != nil || choice.muxed == nil || choice.muxed.ItagNo != 18 {
		t.Errorf("without an AAC track picked %+v, want only muxed 18", choice)
	}

	if choice := pickTrailerFormats(nil); choice.video != nil || choice.audio != nil || choice.muxed != nil {
		t.Errorf("empty list picked %+v", choice)
	}

	noURL := youtube.FormatList{
		{ItagNo: 136, MimeType: `video/mp4; codecs="avc1.4D401F"`, Height: 720},
		{ItagNo: 140, MimeType: `audio/mp4; codecs="mp4a.40.2"`, AudioChannels: 2},
	}
	if choice := pickTrailerFormats(noURL); choice.video != nil || choice.audio != nil {
		t.Errorf("formats without a URL or cipher picked %+v", choice)
	}
}

func TestTrailerError(t *testing.T) {
	for _, err := range []error{youtube.ErrCipherNotFound, youtube.ErrUnexpectedStatusCode(403), youtube.ErrPlayabiltyStatus{Status: "UNPLAYABLE"}} {
		if !errors.Is(trailerError(err), ErrTrailerUnavailable) {
			t.Errorf("trailerError(%v) is not ErrTrailerUnavailable", err)
		}
	}
	for _, err := range []error{youtube.ErrUnexpectedStatusCode(500), context.DeadlineExceeded} {
		if errors.Is(trailerError(err), ErrTrailerUnavailable) {
			t.Errorf("trailerError(%v) wrongly marked unavailable", err)
		}
	}
}

func TestTrailerStreamsNoVideos(t *testing.T) {
	c := testClient(t)
	if _, _, err := c.TrailerStreams(context.Background()); !errors.Is(err, ErrTrailerUnavailable) {
		t.Errorf("no video ids: err = %v", err)
	}
}
