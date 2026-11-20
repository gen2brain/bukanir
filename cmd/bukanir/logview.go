//go:build media

package main

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gen2brain/iup-go/iup"
)

// logRingMax caps how many recent lines the log viewer retains.
const logRingMax = 1000

var logs = &logStore{}

// logCharCount is the character length of the log text widget.
var logCharCount int

type logLine struct {
	text string
	err  bool
}

// logStore captures logger output into a bounded ring, redacting credentials.
type logStore struct {
	mu       sync.Mutex
	lines    []logLine
	partial  string
	redactor *strings.Replacer
	open     bool
	queue    []logLine
	queued   bool
}

// setRedactor installs a replacer that strips the given secret values.
func (s *logStore) setRedactor(secrets []string) {
	pairs := make([]string, 0, len(secrets)*2)
	for _, v := range secrets {
		pairs = append(pairs, v, "***")
	}
	s.mu.Lock()
	if len(pairs) > 0 {
		s.redactor = strings.NewReplacer(pairs...)
	}
	s.mu.Unlock()
}

func (s *logStore) setOpen(v bool) {
	s.mu.Lock()
	s.open = v
	s.queue, s.queued = nil, false
	s.mu.Unlock()
}

// snapshot returns the ring and drops the queued lines, which the ring already holds.
func (s *logStore) snapshot() []logLine {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = nil
	return append([]logLine(nil), s.lines...)
}

// drain takes the lines queued for the viewer.
func (s *logStore) drain() []logLine {
	s.mu.Lock()
	defer s.mu.Unlock()
	q := s.queue
	s.queue, s.queued = nil, false
	return q
}

// redact strips secrets; caller holds the lock.
func (s *logStore) redact(v string) string {
	if s.redactor == nil {
		return v
	}
	return s.redactor.Replace(v)
}

// push appends to the ring; caller holds the lock.
func (s *logStore) push(ln logLine) {
	s.lines = append(s.lines, ln)
	if len(s.lines) > logRingMax {
		s.lines = s.lines[len(s.lines)-logRingMax:]
	}
}

// Write implements io.Writer: split into lines, redact, store, stream to viewer.
func (s *logStore) Write(p []byte) (int, error) {
	s.mu.Lock()
	s.partial += string(p)
	var out []logLine
	for {
		i := strings.IndexByte(s.partial, '\n')
		if i < 0 {
			break
		}
		raw := strings.TrimRight(s.partial[:i], "\r")
		s.partial = s.partial[i+1:]
		if strings.TrimSpace(raw) == "" {
			continue
		}
		ln := logLine{text: s.redact(raw), err: isErrorLine(raw)}
		s.push(ln)
		out = append(out, ln)
	}
	post := false
	if s.open && len(out) > 0 {
		s.queue = append(s.queue, out...)
		post = !s.queued
		s.queued = true
	}
	s.mu.Unlock()

	if post {
		iup.PostMessage(iup.GetHandle("logtext"), "", 0, nil)
	}
	return len(p), nil
}

// isErrorLine reports whether a log line carries the ERROR level marker as a word of its own.
func isErrorLine(line string) bool {
	for _, f := range strings.Fields(line) {
		if f == "ERROR" || f == "ERROR:" {
			return true
		}
	}
	return false
}

// showLogView shows the modeless Log dialog, rebuilt from the captured ring.
func showLogView() {
	dlg := iup.GetHandle("logdlg")
	if dlg == 0 {
		dlg = buildLogView()
	}
	iup.Show(dlg)
	repopulateLog()
	logs.setOpen(true)
}

func buildLogView() iup.Ihandle {
	text := iup.Text().SetHandle("logtext").SetAttributes(map[string]string{
		"MULTILINE":  "YES",
		"READONLY":   "YES",
		"EXPAND":     "YES",
		"WORDWRAP":   "YES",
		"SCROLLBAR":  "VERTICAL",
		"FORMATTING": "YES",
		"FONT":       "Monospace, 9",
	})
	text.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, _ string, _ int, _ any) int {
		appendLogLines(ih, logs.drain())
		return iup.DEFAULT
	}))

	closeBtn := iup.Button("Close").SetAttributes("PADDING=16x4")
	closeBtn.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		logs.setOpen(false)
		iup.Hide(iup.GetHandle("logdlg"))
		return iup.DEFAULT
	}))

	dlg := iup.Dialog(
		iup.Vbox(
			text,
			iup.Hbox(iup.Fill(), closeBtn),
		).SetAttributes("MARGIN=8x8, GAP=8"),
	).SetHandle("logdlg").SetAttributes(map[string]string{
		"TITLE":        "Log",
		"RASTERSIZE":   "700x520",
		"PARENTDIALOG": "dlg",
		"SHRINK":       "YES",
	})
	dlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(iup.Ihandle) int {
		logs.setOpen(false)
		return iup.DEFAULT
	}))
	iup.Map(dlg)
	return dlg
}

// repopulateLog rebuilds the text widget from the current ring.
func repopulateLog() {
	text := iup.GetHandle("logtext")
	if text == 0 {
		return
	}
	text.SetAttribute("REMOVEFORMATTING", "ALL")
	text.SetAttribute("VALUE", "")
	logCharCount = 0
	appendLogLines(text, logs.snapshot())
}

// appendLogLines appends lines with one APPEND, colors each run of error lines and scrolls to the end.
func appendLogLines(text iup.Ihandle, lines []logLine) {
	if len(lines) == 0 {
		return
	}
	var b strings.Builder
	var runs [][2]int
	pos := logCharCount
	if pos > 0 {
		pos++
	}
	for i, ln := range lines {
		if i > 0 {
			b.WriteByte('\n')
			pos++
		}
		b.WriteString(ln.text)
		end := pos + utf8.RuneCountInString(ln.text)
		if ln.err {
			if n := len(runs); n > 0 && runs[n-1][1]+1 == pos {
				runs[n-1][1] = end
			} else {
				runs = append(runs, [2]int{pos, end})
			}
		}
		pos = end
	}
	text.SetAttribute("APPEND", b.String())
	logCharCount = pos
	for _, r := range runs {
		tag := iup.User()
		tag.SetAttributes(fmt.Sprintf(`SELECTIONPOS="%d:%d", FGCOLOR="255 85 85"`, r[0], r[1]))
		iup.SetAttributeHandle(text, "ADDFORMATTAG", tag)
	}
	text.SetAttribute("SCROLLTOPOS", fmt.Sprintf("%d", pos))
}
