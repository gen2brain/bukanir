package bukanir

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/proxy"
)

func TestTor(t *testing.T) {
	time.Sleep(1 * time.Second)

	defer func() {
		err := ttor.Stop()
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(2 * time.Second)

		if ttor.Running() {
			t.Fatal("tor still running")
		}
	}()

	if !ttor.Running() {
		t.Fatal("tor not running")
	}

	if !ttor.ControlRunning() {
		t.Fatal("tor control not running")
	}

	ipCurrent, err := getIP(ttor.Port)
	if err != nil {
		t.Fatal(err)
	}

	ttor.Renew()
	time.Sleep(1 * time.Second)

	ipNew, err := getIP(ttor.Port)
	if err != nil {
		t.Fatal(err)
	}

	if ipCurrent == ipNew {
		t.Fatal("renew failed")
	}
}

func getIP(port string) (ip string, e error) {
	proxyURL, err := url.Parse("socks5://127.0.0.1:" + port)
	if err != nil {
		e = fmt.Errorf("failed to parse proxy URL: %v\n", err)
		return
	}

	dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		e = fmt.Errorf("failed to obtain proxy dialer: %v\n", err)
		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", "http://canihazip.com/s", nil)
	if err != nil {
		e = err
		return
	}

	res, err := client.Do(req)
	if err != nil {
		e = err
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		e = err
		return
	}

	res.Body.Close()

	b := strings.Replace(string(body), "\n", "", -1)

	i := net.ParseIP(b)
	if i == nil {
		e = errors.New("ip is nil")
		return
	}

	ip = i.String()
	return
}
