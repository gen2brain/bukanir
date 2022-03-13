//go:build !torcmd
// +build !torcmd

package bukanir

import (
	"os"
	"strconv"
	"runtime"

	libtor "github.com/gen2brain/go-libtor"
	bine "github.com/cretz/bine/tor"
)

// tor type
type tor struct {
	User     string
	Port     string
	CtrlPort string
	DataDir  string
	Command  *bine.Tor
}

// Exists checks if Tor binary exists
func (t *tor) Exists() bool {
	return true
}

// Start starts Tor
func (t *tor) Start() error {
	ctrl, _ := strconv.Atoi(t.CtrlPort)
	conf := &bine.StartConf{
		//NoHush:                 true,
		//DebugWriter:            os.Stderr,
		ProcessCreator:         libtor.Creator,
		DataDir:                t.DataDir,
		ControlPort:            ctrl,
		EnableNetwork:          true,
		DisableCookieAuth:      true,
		DisableEagerAuth:       true,
		NoAutoSocksPort:        true,
		UseEmbeddedControlConn: runtime.GOOS == "linux",
		ExtraArgs:              []string{"--SocksPort", t.Port, "--quiet"},
	}

	b, err := bine.Start(nil, conf)
	if err != nil {
		return err
	}

	t.Command = b
	return nil
}

// Stop stops Tor
func (t *tor) Stop() error {
	defer os.RemoveAll(t.DataDir)
	if t.Command == nil {
		return nil
	}

	return t.Command.Close()
}
