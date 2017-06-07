package bukanir

// libtorrent-based torrent client that allows to download torrents and stream it through HTTP
// Based on https://github.com/steeve/torrent2http with added modifications from https://github.com/anteo/torrent2http

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	lt "github.com/gen2brain/libtorrent-go"
)

type FileStatusInfo struct {
	Name     string  `json:"name"`
	SavePath string  `json:"save_path"`
	Url      string  `json:"url"`
	Size     int64   `json:"size"`
	Offset   int64   `json:"offset"`
	Download int64   `json:"download"`
	Progress float32 `json:"progress"`
}

type LsInfo struct {
	Files []FileStatusInfo `json:"files"`
}

type SessionStatus struct {
	Name          string  `json:"name"`
	State         int     `json:"state"`
	StateStr      string  `json:"state_str"`
	Error         string  `json:"error"`
	Progress      float32 `json:"progress"`
	DownloadRate  float32 `json:"download_rate"`
	UploadRate    float32 `json:"upload_rate"`
	TotalDownload int64   `json:"total_download"`
	TotalUpload   int64   `json:"total_upload"`
	NumPeers      int     `json:"num_peers"`
	NumSeeds      int     `json:"num_seeds"`
	TotalSeeds    int     `json:"total_seeds"`
	TotalPeers    int     `json:"total_peers"`
}

type Config struct {
	Uri                 string `json:"uri"`
	BindAddress         string `json:"bind_address"`
	FileIndex           int    `json:"file_index"`
	MaxUploadRate       int    `json:"max_upload_rate"`
	MaxDownloadRate     int    `json:"max_download_rate"`
	DownloadPath        string `json:"download_path"`
	UserAgent           string `json:"user_agent"`
	KeepFiles           bool   `json:"keep_files"`
	Encryption          int    `json:"encryption"`
	NoSparseFile        bool   `json:"no_sparse_file"`
	PeerConnectTimeout  int    `json:"peer_connect_timeout"`
	RequestTimeout      int    `json:"request_timeout"`
	TorrentConnectBoost int    `json:"torrent_connect_boost"`
	ConnectionSpeed     int    `json:"connection_speed"`
	ListenPort          int    `json:"listen_port"`
	MinReconnectTime    int    `json:"min_reconnect_time"`
	MaxFailCount        int    `json:"max_fail_count"`
	RandomPort          bool   `json:"random_port"`
	DhtRouters          string `json:"dht_routers"`
	Trackers            string `json:"trackers"`
	Proxy               bool   `json:"proxy"`
	ProxyHost           string `json:"proxy_host"`
	ProxyPort           int    `json:"proxy_port"`
	Blocklist           bool   `json:"blocklist"`
	Verbose             bool   `json:"verbose"`
}

const (
	LibName    = "libtorrent-rasterbar"
	LibVersion = lt.LIBTORRENT_VERSION
)

type torrent struct {
	config        Config
	session       lt.Session
	torrent       lt.TorrentHandle
	torrentFs     *torrentFS
	forceShutdown chan bool
	httpListener  net.Listener
}

const (
	stateQueuedForChecking = iota
	stateCheckingFiles
	stateDownloadingMetadata
	stateDownloading
	stateFinished
	stateSeeding
	stateAllocating
	stateCheckingResumeData
)

var stateStrings = map[int]string{
	stateQueuedForChecking:   "Queued",
	stateCheckingFiles:       "Checking",
	stateDownloadingMetadata: "Downloading torrent metadata",
	stateDownloading:         "Downloading",
	stateFinished:            "Finished",
	stateSeeding:             "Seeding",
	stateAllocating:          "Allocating",
	stateCheckingResumeData:  "Checking resume data",
}

func (t *torrent) popAlert() lt.Alert {
	alert := t.session.PopAlert()
	if alert.Swigcptr() == 0 {
		return nil
	}
	return alert
}

func (t *torrent) consumeAlerts() {
	for {
		var alert lt.Alert
		if alert = t.popAlert(); alert == nil {
			break
		}
		lt.DeleteAlert(alert)
	}
}

func (t *torrent) buildTorrentParams(uri string) lt.AddTorrentParams {
	fileUri, err := url.Parse(uri)
	if err != nil {
		log.Printf("ERROR: url.Parse: %v", err)
	}

	torrentParams := lt.NewAddTorrentParams()

	error := lt.NewErrorCode()
	defer lt.DeleteErrorCode(error)

	if err != nil {
		log.Printf("ERROR: buildTorrentParams: %v", err)
	}

	if fileUri.Scheme == "file" {
		uriPath := fileUri.Path
		if uriPath != "" && runtime.GOOS == "windows" && os.IsPathSeparator(uriPath[0]) {
			uriPath = uriPath[1:]
		}
		absPath, err := filepath.Abs(uriPath)
		if err != nil {
			log.Printf("ERROR: buildTorrentParams: %v", err.Error())
		}
		if t.config.Verbose {
			log.Printf("T2HTTP: Opening local file: %s", absPath)
		}
		if _, err := os.Stat(absPath); err != nil {
			log.Printf("ERROR: buildTorrentParams: %v", err.Error())
		}

		torrentInfo := lt.NewTorrentInfo(absPath, error)
		if error.Value() != 0 {
			log.Printf("ERROR: buildTorrentParams: %v", error.Message())
		}
		defer lt.DeleteTorrentInfo(torrentInfo)

		torrentParams.SetTorrentInfo(torrentInfo)
	} else {
		if t.config.Verbose {
			log.Printf("T2HTTP: Will fetch: %s", uri)
		}
		torrentParams.SetUrl(uri)
	}

	if t.config.Verbose {
		log.Printf("T2HTTP: Setting save path: %s", t.config.DownloadPath)
	}
	torrentParams.SetSavePath(t.config.DownloadPath)

	if t.config.NoSparseFile {
		if t.config.Verbose {
			log.Println("T2HTTP: Disabling sparse file support...")
		}
		torrentParams.SetStorageMode(lt.StorageModeCompact)
	}

	return torrentParams
}

func (t *torrent) addTorrent(torrentParams lt.AddTorrentParams) {
	if t.config.Verbose {
		log.Println("T2HTTP: Adding torrent")
	}

	error := lt.NewErrorCode()
	defer lt.DeleteErrorCode(error)

	t.torrent = t.session.AddTorrent(torrentParams, error)
	if error.Value() != 0 {
		log.Printf("ERROR: addTorrent: %v", error.Message())
	}

	defer lt.DeleteAddTorrentParams(torrentParams)

	if t.config.Verbose {
		log.Println("T2HTTP: Enabling sequential download")
	}
	t.torrent.SetSequentialDownload(true)

	if t.config.Trackers != "" {
		trackers := strings.Split(t.config.Trackers, ",")
		startTier := 256 - len(trackers)
		for n, tracker := range trackers {
			tracker = strings.TrimSpace(tracker)
			announceEntry := lt.NewAnnounceEntry(tracker)
			announceEntry.SetTier(byte(startTier + n))
			if t.config.Verbose {
				log.Printf("T2HTTP: Adding tracker: %s", tracker)
			}
			t.torrent.AddTracker(announceEntry)
			lt.DeleteAnnounceEntry(announceEntry)
		}
	}

	if t.config.Verbose {
		log.Printf("T2HTTP: Downloading torrent: %s", t.torrent.Status().GetName())
	}
	t.torrentFs = newTorrentFS(t.torrent, t.config.FileIndex)
}

func (t *torrent) removeTorrent() {
	var flag int
	state := t.torrent.Status().GetState()
	if state != stateCheckingFiles && state != stateQueuedForChecking && !t.config.KeepFiles {
		flag = int(lt.SessionDeleteFiles)
	}

	if t.config.Verbose {
		log.Println("T2HTTP: Removing the torrent")
	}
	t.session.RemoveTorrent(t.torrent, flag)
}

func (t *torrent) startSession() {
	if t.config.Verbose {
		log.Println("T2HTTP: Starting session...")
		log.Println(fmt.Sprintf("T2HTTP: Library %s-%s", LibName, LibVersion))
	}

	t.session = lt.NewSession(
		lt.NewFingerprint("LT", lt.LIBTORRENT_VERSION_MAJOR, lt.LIBTORRENT_VERSION_MINOR, 0, 0),
		int(lt.SessionAddDefaultPlugins),
	)

	alertMask := uint(lt.AlertErrorNotification) | uint(lt.AlertStorageNotification) |
		uint(lt.AlertTrackerNotification) | uint(lt.AlertStatusNotification)

	t.session.SetAlertMask(alertMask)

	settings := t.session.Settings()

	settings.SetStrictEndGameMode(true)
	settings.SetAnnounceToAllTrackers(true)
	settings.SetAnnounceToAllTiers(true)
	settings.SetAnnounceDoubleNat(true)

	settings.SetRequestTimeout(t.config.RequestTimeout)
	settings.SetPeerConnectTimeout(t.config.PeerConnectTimeout)
	settings.SetTorrentConnectBoost(t.config.TorrentConnectBoost)
	settings.SetConnectionSpeed(t.config.ConnectionSpeed)
	settings.SetMinReconnectTime(t.config.MinReconnectTime)
	settings.SetMaxFailcount(t.config.MaxFailCount)

	//settings.SetRateLimitIpOverhead(true)
	settings.SetNoAtimeStorage(true)
	settings.SetPrioritizePartialPieces(false)
	//settings.SetFreeTorrentHashes(true)
	//settings.SetUseParoleMode(true)
	settings.SetMinAnnounceInterval(60)
	settings.SetTrackerBackoff(0)

	settings.SetLowPrioDisk(false)
	settings.SetLockDiskCache(true)
	settings.SetDiskCacheAlgorithm(lt.SessionSettingsLru)
	//settings.SetDiskCacheAlgorithm(lt.SessionSettingsLargestContiguous)
	settings.SetSeedChokingAlgorithm(int(lt.SessionSettingsFastestUpload))

	settings.SetUpnpIgnoreNonrouters(true)
	settings.SetLazyBitfields(true)
	settings.SetStopTrackerTimeout(1)
	settings.SetAutoScrapeInterval(1200)
	settings.SetAutoScrapeMinInterval(900)
	//settings.SetRateLimitUtp(true)
	settings.SetMixedModeAlgorithm(int(lt.SessionSettingsPreferTcp))

	settings.SetConnectionsLimit(100 * runtime.NumCPU())

	t.session.SetSettings(settings)

	err := lt.NewErrorCode()
	defer lt.DeleteErrorCode(err)

	rand.Seed(time.Now().UnixNano())
	portLower := t.config.ListenPort
	if t.config.RandomPort {
		portLower = rand.Intn(6999-6881) + 6881
	}
	portUpper := portLower + 10

	ports := lt.NewStd_pair_int_int(portLower, portUpper)
	defer lt.DeleteStd_pair_int_int(ports)

	t.session.ListenOn(ports, err)
	if err.Value() != 0 {
		log.Printf("ERROR: startSession: %v", err.Message())
	}

	settings = t.session.Settings()
	if t.config.UserAgent != "" {
		settings.SetUserAgent(t.config.UserAgent)
	} else {
		settings.SetUserAgent(fmt.Sprintf("%s-%s", LibName, LibVersion))
	}
	if t.config.MaxDownloadRate >= 0 {
		settings.SetDownloadRateLimit(t.config.MaxDownloadRate * 1024)
	}
	if t.config.MaxUploadRate >= 0 {
		settings.SetUploadRateLimit(t.config.MaxUploadRate * 1024)
	}

	settings.SetEnableIncomingTcp(true)
	settings.SetEnableOutgoingTcp(true)
	settings.SetEnableIncomingUtp(true)
	settings.SetEnableOutgoingUtp(true)

	if t.config.Proxy {
		proxySettings := lt.NewProxySettings()
		proxySettings.SetHostname(t.config.ProxyHost)
		proxySettings.SetPort(uint16(t.config.ProxyPort))
		proxySettings.SetType(byte(lt.ProxySettingsSocks5))
		proxySettings.SetProxyHostnames(false)
		proxySettings.SetProxyPeerConnections(true)

		t.session.SetProxy(proxySettings)
		t.session.SetPeerProxy(proxySettings)
		t.session.SetTrackerProxy(proxySettings)
		t.session.SetWebSeedProxy(proxySettings)
		t.session.SetDhtProxy(proxySettings)

		settings.SetForceProxy(false)
	}

	if t.config.Blocklist {
		if t.config.Verbose {
			log.Println("T2HTTP: Setting blocklist ip filter")
		}

		t.setIpFilter()

		settings.SetApplyIpFilterToTrackers(false)
	}

	t.session.SetSettings(settings)

	if t.config.DhtRouters != "" {
		routers := strings.Split(t.config.DhtRouters, ",")
		for _, router := range routers {
			router = strings.TrimSpace(router)
			if len(router) != 0 {
				var err error
				hostPort := strings.SplitN(router, ":", 2)
				host := strings.TrimSpace(hostPort[0])
				port := 6881
				if len(hostPort) > 1 {
					port, err = strconv.Atoi(strings.TrimSpace(hostPort[1]))
					if err != nil {
						log.Printf("ERROR: startSession: %v", err)
					}
				}

				bind := lt.NewStd_pair_string_int(host, port)
				defer lt.DeleteStd_pair_string_int(bind)

				t.session.AddDhtRouter(bind)
				if t.config.Verbose {
					log.Printf("T2HTTP: Added DHT router: %s:%d", host, port)
				}
			}
		}
	}

	if t.config.Verbose {
		log.Println("T2HTTP: Setting encryption settings")
	}

	encryptionSettings := lt.NewPeSettings()
	defer lt.DeletePeSettings(encryptionSettings)

	encryptionSettings.SetOutEncPolicy(byte(lt.LibtorrentPe_settingsEnc_policy(t.config.Encryption)))
	encryptionSettings.SetInEncPolicy(byte(lt.LibtorrentPe_settingsEnc_policy(t.config.Encryption)))
	encryptionSettings.SetAllowedEncLevel(byte(lt.PeSettingsBoth))
	encryptionSettings.SetPreferRc4(true)

	t.session.SetPeSettings(encryptionSettings)
}

func (t *torrent) setIpFilter() {
	file := bytes.NewReader(ipFilterDecode())

	gz, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("ERROR: setIpFilter %v", err)
		return
	}

	defer gz.Close()

	filter := lt.NewIpFilter()

	scanner := bufio.NewScanner(gz)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "-")
		if len(line) == 2 {
			filter.AddRule(lt.AddressFromString(line[0]), lt.AddressFromString(line[1]), int(lt.IpFilterBlocked))
		}
	}

	t.session.SetIpFilter(filter)
}

func (t *torrent) startServices() {
	if t.config.Verbose {
		log.Println("T2HTTP: Starting DHT...")
	}
	t.session.StartDht()

	if t.config.Verbose {
		log.Println("T2HTTP: Starting LSD...")
	}
	t.session.StartLsd()

	if t.config.Verbose {
		log.Println("T2HTTP: Starting UPNP...")
	}
	t.session.StartUpnp()

	if t.config.Verbose {
		log.Println("T2HTTP: Starting NATPMP...")
	}
	t.session.StartNatpmp()
}

func (t *torrent) stopServices() {
	if t.config.Verbose {
		log.Println("T2HTTP: Stopping DHT...")
	}
	t.session.StopDht()

	if t.config.Verbose {
		log.Println("T2HTTP: Stopping LSD...")
	}
	t.session.StopLsd()

	if t.config.Verbose {
		log.Println("T2HTTP: Stopping UPNP...")
	}
	t.session.StopUpnp()

	if t.config.Verbose {
		log.Println("T2HTTP: Stopping NATPMP...")
	}
	t.session.StopNatpmp()
}

func (t *torrent) startHTTP() {
	if t.config.Verbose {
		log.Println("T2HTTP: Starting HTTP Server...")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", t.statusHandler)
	mux.HandleFunc("/ls", t.lsHandler)
	mux.Handle("/get/", http.StripPrefix("/get/", t.getHandler(http.FileServer(t.torrentFs))))
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		t.Stop()
		fmt.Fprintf(w, "OK")
	})
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(t.torrentFs)))

	handler := http.Handler(mux)

	if t.config.Verbose {
		log.Printf("T2HTTP: Listening HTTP on %s...\n", t.config.BindAddress)
	}
	s := &http.Server{
		Addr:    t.config.BindAddress,
		Handler: handler,
	}

	var err error
	t.httpListener, err = net.Listen("tcp4", t.config.BindAddress)
	if err != nil {
		log.Printf("ERROR: startHTTP: %v", err)
	} else {
		go s.Serve(t.httpListener)
	}
}

func (t *torrent) stopHTTP() {
	if t.httpListener != nil {
		err := t.httpListener.Close()
		if err != nil {
			log.Printf("ERROR: stopHTTP: %v", err)
		}
		t.httpListener = nil
	}
}

func (t *torrent) statusHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, _ := t.Status()
	w.Write([]byte(status))
}

func (t *torrent) lsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	files, _ := t.Ls()
	w.Write([]byte(files))
}

func (t *torrent) getHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index, err := strconv.Atoi(r.URL.String())
		if err == nil && t.torrentFs.HasTorrentInfo() {
			file, err := t.torrentFs.FileAt(index)
			if err == nil {
				r.URL.Path = file.Name()
				h.ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})
}

func (t *torrent) loop() {
	t.forceShutdown = make(chan bool, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-t.forceShutdown:
			t.stopHTTP()
			if t.config.Verbose {
				log.Println("T2HTTP: Exit loop")
			}
			return
		case <-signalChan:
			t.forceShutdown <- true
		case <-time.After(500 * time.Millisecond):
			t.consumeAlerts()
			t.torrentFs.LoadFileProgress()
			if os.Getppid() == 1 {
				t.forceShutdown <- true
			}
		}
	}
}

func (t *torrent) Startup(cfg string) {
	err := json.Unmarshal([]byte(cfg), &t.config)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
	}

	t.startSession()
	t.startServices()
	t.addTorrent(t.buildTorrentParams(t.config.Uri))
	t.startHTTP()
	t.loop()
}

func (t *torrent) Shutdown() {
	if t.config.Verbose {
		log.Println("T2HTTP: Shutdown torrentFs...")
	}
	t.torrentFs.Shutdown()

	if t.session != nil {
		if t.torrent != nil {
			t.removeTorrent()
		}
		if t.config.Verbose {
			log.Println("T2HTTP: Aborting the session")
		}
		lt.DeleteSession(t.session)
	}
}

func (t *torrent) Stop() {
	t.forceShutdown <- true
	t.Shutdown()
}

func (t *torrent) Running() bool {
	return t.httpListener != nil
}

func (t *torrent) Status() (string, error) {
	var status SessionStatus
	if t.torrent == nil {
		status = SessionStatus{State: -1}
	} else {
		tstatus := t.torrent.Status()
		status = SessionStatus{
			Name:          tstatus.GetName(),
			State:         int(tstatus.GetState()),
			StateStr:      stateStrings[int(tstatus.GetState())],
			Error:         tstatus.GetError(),
			Progress:      tstatus.GetProgress(),
			TotalDownload: tstatus.GetTotalDownload(),
			TotalUpload:   tstatus.GetTotalUpload(),
			DownloadRate:  float32(tstatus.GetDownloadRate()) / 1024,
			UploadRate:    float32(tstatus.GetUploadRate()) / 1024,
			NumPeers:      tstatus.GetNumPeers(),
			TotalPeers:    tstatus.GetNumIncomplete(),
			NumSeeds:      tstatus.GetNumSeeds(),
			TotalSeeds:    tstatus.GetNumComplete()}
	}

	js, err := json.MarshalIndent(status, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}

func (t *torrent) Ls() (string, error) {
	retFiles := LsInfo{}

	if t.torrentFs.HasTorrentInfo() {
		for _, file := range t.torrentFs.Files() {
			url := url.URL{
				Scheme: "http",
				Host:   t.config.BindAddress,
				Path:   "/files/" + file.Name(),
			}
			fi := FileStatusInfo{
				Name:     file.Name(),
				Size:     file.Size(),
				Offset:   file.Offset(),
				Download: file.Downloaded(),
				Progress: file.Progress(),
				SavePath: file.SavePath(),
				Url:      url.String(),
			}
			retFiles.Files = append(retFiles.Files, fi)
		}
	}

	js, err := json.MarshalIndent(retFiles, "", "    ")
	if err != nil {
		return "empty", err
	}

	return string(js[:]), nil
}
