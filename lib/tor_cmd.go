//go:build torcmd
// +build torcmd

package bukanir

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// tor type
type tor struct {
	User     string
	Port     string
	CtrlPort string
	DataDir  string
	Command  *exec.Cmd
}

// Exists checks if Tor binary exists
func (t *tor) Exists() bool {
	_, err := exec.LookPath("tor")
	if err == nil {
		return true
	}

	if runtime.GOOS == "windows" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Printf("ERROR: %v\n", err)
		}

		tor := filepath.Join(dir, "tor.exe")
		if _, err := os.Stat(tor); err == nil {
			return true
		}
	}

	return false
}

// Start starts Tor
func (t *tor) Start() error {
	if runtime.GOOS == "windows" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}

		tor := filepath.Join(dir, "tor.exe")

		t.Command = exec.Command(tor, "--SocksPort", t.Port, "--DataDirectory", t.DataDir, "--ControlPort", t.CtrlPort, "-f", "nonexistenttorrc", "--ignore-missing-torrc")
		if verbose {
			log.Printf("BUK: %s\n", strings.Join(t.Command.Args, " "))
		}

		err = t.Command.Start()
		return err
	} else if runtime.GOOS != "android" {
		c := fmt.Sprintf("tor --user %s --SocksPort %s --DataDirectory %s --ControlPort %s -f nonexistenttorrc --ignore-missing-torrc", t.User, t.Port, t.DataDir, t.CtrlPort)
		if verbose {
			log.Printf("BUK: %s\n", c)
		}

		t.Command = exec.Command("sh", "-c", c)

		err := t.Command.Start()
		return err
	}

	return nil
}

// Stop stops Tor
func (t *tor) Stop() error {
	if t.Command == nil {
		return nil
	}

	defer os.RemoveAll(t.DataDir)

	err := t.Command.Process.Kill()
	if err != nil {
		return t.Command.Process.Kill()
	}

	return nil
}
