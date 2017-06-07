package bukanir

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
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

// NewTor returns new tor
func NewTor(user, port, ctrlPort, dataDir string) *tor {
	t := &tor{}
	t.User = user
	t.Port = port
	t.CtrlPort = ctrlPort
	t.DataDir = dataDir

	return t
}

// Exists checks if Tor binary exists
func (t *tor) Exists() bool {
	if which("tor") != "" {
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

// Running checks if Tor is running
func (t *tor) Running() bool {
	_, err := net.Dial("tcp", "127.0.0.1:"+t.Port)
	if err == nil {
		return true
	}

	return false
}

// ControlRunning checks if Tor is running on control port
func (t *tor) ControlRunning() bool {
	_, err := net.Dial("tcp", "127.0.0.1:"+t.CtrlPort)
	if err == nil {
		return true
	}

	return false
}

// SetDataDir sets datadir ownership/permissions
func (t *tor) SetDataDir() error {
	usr, err := user.Lookup(t.User)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(usr.Gid)
	if err != nil {
		return err
	}

	err = os.Chown(t.DataDir, uid, gid)
	if err != nil {
		return err
	}

	err = os.Chmod(t.DataDir, 0777)
	if err != nil {
		return err
	}

	return nil
}

// Start starts Tor
func (t *tor) Start() error {
	if runtime.GOOS == "windows" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}

		tor := filepath.Join(dir, "tor.exe")

		t.Command = exec.Command(tor, "--SocksPort", t.Port, "--DataDirectory", t.DataDir, "--ControlPort", t.CtrlPort)
		if verbose {
			log.Printf("BUK: %s\n", strings.Join(t.Command.Args, " "))
		}

		err = t.Command.Start()
		return err
	} else if runtime.GOOS != "android" {
		c := fmt.Sprintf("tor --user %s --SocksPort %s --DataDirectory %s --ControlPort %s", t.User, t.Port, t.DataDir, t.CtrlPort)
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

	os.RemoveAll(t.DataDir)

	err := t.Command.Process.Kill()
	if err != nil {
		return t.Command.Process.Kill()
	}

	return nil
}

// Renew renews IP address
func (t *tor) Renew() {
	conn, err := net.Dial("tcp", "127.0.0.1:"+t.CtrlPort)
	defer conn.Close()

	if err != nil {
		log.Printf("ERROR: %v\n", err.Error())
		return
	}

	var n int
	var buff []byte

	conn.Write([]byte("AUTHENTICATE\r\n"))

	buff = make([]byte, 1024)
	n, err = conn.Read(buff)
	if err != nil {
		log.Printf("ERROR: %v\n", err.Error())
	}

	if strings.HasPrefix(string(buff[:n]), "250") {
		conn.Write([]byte("SIGNAL NEWNYM\r\n"))

		buff = make([]byte, 1024)
		n, err = conn.Read(buff)
		if err != nil {
			log.Printf("ERROR: %v\n", err.Error())
		}

		if !strings.HasPrefix(string(buff[:n]), "250") {
			log.Printf("ERROR: %s\n", string(buff[:n]))
		}

		if verbose {
			log.Printf("BUK: %s", strings.Replace(string(buff[:n]), "\n", "", -1))
		}

	} else {
		log.Printf("ERROR: %s\n", string(buff[:n]))
	}
}
