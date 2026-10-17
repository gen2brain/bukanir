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
}

var logs = &logStore{}

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
	s.mu.Unlock()
}

func (s *logStore) snapshot() []logLine {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]logLine(nil), s.lines...)
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
		ln := logLine{text: s.redact(raw), err: strings.Contains(strings.ToLower(raw), "error")}
		s.push(ln)
		out = append(out, ln)
	}
	open := s.open
	s.mu.Unlock()

	if open {
		for _, ln := range out {
			postLogLine(ln)
		}
	}
	return len(p), nil
}

// postLogLine hands a line to the viewer's text widget (err flag in the code).
func postLogLine(ln logLine) {
	code := 0
	if ln.err {
		code = 1
	}
	iup.PostMessage(iup.GetHandle("logtext"), ln.text, code, nil)
}

// logCharCount is the character length of the log text widget.
var logCharCount int

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
	text.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, line string, code int, _ any) int {
		appendLogText(ih, line, code == 1)
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
	for _, ln := range logs.snapshot() {
		appendLogText(text, ln.text, ln.err)
	}
}

// appendLogText appends one line (red when an error) and scrolls to it.
func appendLogText(text iup.Ihandle, line string, isErr bool) {
	start := logCharCount
	if logCharCount > 0 {
		start++
	}
	text.SetAttribute("APPEND", line)
	end := start + utf8.RuneCountInString(line)
	logCharCount = end
	if isErr {
		tag := iup.User()
		tag.SetAttributes(fmt.Sprintf(`SELECTIONPOS="%d:%d", FGCOLOR="255 85 85"`, start, end))
		iup.SetAttributeHandle(text, "ADDFORMATTAG", tag)
	}
	text.SetAttribute("SCROLLTOPOS", fmt.Sprintf("%d", end))
}
