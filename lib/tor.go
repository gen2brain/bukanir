package bukanir

import (
	"net"
	"os"
	"os/user"
	"strconv"
)

// NewTor returns new tor
func NewTor(user, port, ctrlPort, dataDir string) *tor {
	t := &tor{}
	t.User = user
	t.Port = port
	t.CtrlPort = ctrlPort
	t.DataDir = dataDir

	return t
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

	err = os.Chmod(t.DataDir, 0700)
	if err != nil {
		return err
	}

	return nil
}
