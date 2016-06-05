package torrent2http

import (
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

	lt "libtorrent-go"
)

type fileStatusInfo struct {
	Name     string  `json:"name"`
	SavePath string  `json:"save_path"`
	Url      string  `json:"url"`
	Size     int64   `json:"size"`
	Offset   int64   `json:"offset"`
	Download int64   `json:"download"`
	Progress float32 `json:"progress"`
}

type lsInfo struct {
	Files []fileStatusInfo `json:"files"`
}

type sessionStatus struct {
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
	KeepComplete        bool   `json:"keep_complete"`
	KeepIncomplete      bool   `json:"keep_incomplete"`
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
	Verbose             bool   `json:"verbose"`
}

const (
	LibName    = "libtorrent-rasterbar"
	LibVersion = lt.LIBTORRENT_VERSION
)

var (
	config        Config
	session       lt.Session
	sessionHandle lt.SessionHandle
	torrent       lt.TorrentHandle
	torrentFs     *torrentFS
	forceShutdown chan bool
	httpListener  net.Listener
)

const (
	state_queued_for_checking = iota
	state_checking_files
	state_downloading_metadata
	state_downloading
	state_finished
	state_seeding
	state_allocating
	state_checking_resume_data
)

var stateStrings = map[int]string{
	state_queued_for_checking:  "Queued",
	state_checking_files:       "Checking",
	state_downloading_metadata: "Downloading metadata",
	state_downloading:          "Downloading",
	state_finished:             "Finished",
	state_seeding:              "Seeding",
	state_allocating:           "Allocating",
	state_checking_resume_data: "Checking resume data",
}

func popAlert() lt.Alert {
	alert := sessionHandle.PopAlert()
	if alert.Swigcptr() == 0 {
		return nil
	}
	return alert
}

func consumeAlerts() {
	for {
		var alert lt.Alert
		if alert = popAlert(); alert == nil {
			break
		}
	}
}

func waitForAlert(name string, timeout time.Duration) lt.Alert {
	start := time.Now()
	for {
		for {
			alert := sessionHandle.WaitForAlert(lt.Milliseconds__SWIG_0(100))
			if time.Now().Sub(start) > timeout {
				return nil
			}
			if alert.Swigcptr() != 0 {
				alert = popAlert()
				if alert.What() == name {
					return alert
				}
			}
		}
	}
}

func filesToRemove() []string {
	var files []string
	if torrentFs.HasTorrentInfo() {
		for _, file := range torrentFs.Files() {
			if (!config.KeepComplete || !file.IsComplete()) && (!config.KeepIncomplete || file.IsComplete()) {
				if _, err := os.Stat(file.SavePath()); !os.IsNotExist(err) {
					files = append(files, file.SavePath())
				}
			}
		}
	}
	return files
}

func removeFiles(files []string) {
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			log.Printf("Error removeFiles: %v", err)
		} else {
			// Remove empty folders as well
			path := filepath.Dir(file)
			separator := fmt.Sprintf("%c", filepath.Separator)
			savePath, _ := filepath.Abs(config.DownloadPath)
			savePath = strings.TrimSuffix(savePath, separator)
			for path != savePath {
				os.Remove(path)
				path = strings.TrimSuffix(filepath.Dir(path), separator)
			}
		}
	}
}

func buildTorrentParams(uri string) lt.AddTorrentParams {
	fileUri, err := url.Parse(uri)
	torrentParams := lt.NewAddTorrentParams()
	error := lt.NewErrorCode()
	if err != nil {
		log.Printf("Error buildTorrentParams: %v", err)
	}
	if fileUri.Scheme == "file" {
		uriPath := fileUri.Path
		if uriPath != "" && runtime.GOOS == "windows" && os.IsPathSeparator(uriPath[0]) {
			uriPath = uriPath[1:]
		}
		absPath, err := filepath.Abs(uriPath)
		if err != nil {
			log.Printf("Error buildTorrentParams: %v", err.Error())
		}
		if config.Verbose {
			log.Printf("Opening local file: %s", absPath)
		}
		if _, err := os.Stat(absPath); err != nil {
			log.Printf("Error buildTorrentParams: %v", err.Error())
		}
		torrentInfo := lt.NewTorrentInfo(absPath, error)
		if error.Value() != 0 {
			log.Printf("Error buildTorrentParams: %v", error.Message())
		}
		torrentParams.SetTorrentInfo(torrentInfo)
	} else {
		if config.Verbose {
			log.Printf("Will fetch: %s", uri)
		}
		torrentParams.SetUrl(uri)
	}

	if config.Verbose {
		log.Printf("Setting save path: %s", config.DownloadPath)
	}
	torrentParams.SetSavePath(config.DownloadPath)

	if config.NoSparseFile {
		if config.Verbose {
			log.Println("Disabling sparse file support...")
		}
		//torrentParams.SetStorageMode(lt.StorageModeCompact)
		torrentParams.SetStorageMode(lt.StorageModeAllocate)
	}

	return torrentParams
}

func addTorrent(torrentParams lt.AddTorrentParams) {
	if config.Verbose {
		log.Println("Adding torrent")
	}
	error := lt.NewErrorCode()
	torrent = sessionHandle.AddTorrent(torrentParams, error)
	if error.Value() != 0 {
		log.Printf("Error addTorrent: %v", error.Message())
	}

	if config.Verbose {
		log.Println("Enabling sequential download")
	}
	torrent.SetSequentialDownload(true)

	if config.Trackers != "" {
		trackers := strings.Split(config.Trackers, ",")
		startTier := 256 - len(trackers)
		for n, tracker := range trackers {
			tracker = strings.TrimSpace(tracker)
			announceEntry := lt.NewAnnounceEntry(tracker)
			announceEntry.SetTier(byte(startTier + n))
			if config.Verbose {
				log.Printf("Adding tracker: %s", tracker)
			}
			torrent.AddTracker(announceEntry)
		}
	}

	if config.Verbose {
		log.Printf("Downloading torrent: %s", torrent.Status().GetName())
	}
	torrentFs = newTorrentFS(torrent, config.FileIndex)
}

func removeTorrent() {
	var flag int
	var files []string

	state := torrent.Status().GetState()
	if state != state_checking_files && state != state_queued_for_checking && !config.KeepFiles {
		if !config.KeepComplete && !config.KeepIncomplete {
			flag = int(lt.SessionHandleDeleteFiles)
		} else {
			files = filesToRemove()
		}
	}
	if config.Verbose {
		log.Println("Removing the torrent")
	}
	sessionHandle.RemoveTorrent(torrent, flag)
	if flag != 0 || len(files) > 0 {
		if config.Verbose {
			log.Println("Waiting for files to be removed")
		}
		waitForAlert("cache_flushed_alert", 10*time.Second)
		removeFiles(files)
	}
}

func startSession() {
	if config.Verbose {
		log.Println(fmt.Sprintf("%s-%s", LibName, LibVersion))
		log.Println("Starting session...")
	}

	session = lt.NewSession(
		lt.NewFingerprint("LT", lt.LIBTORRENT_VERSION_MAJOR, lt.LIBTORRENT_VERSION_MINOR, 0, 0),
		int(lt.SessionHandleAddDefaultPlugins),
	)

	sessionHandle = session.GetHandle()

	alertMask := uint(lt.AlertErrorNotification) | uint(lt.AlertStorageNotification) |
		uint(lt.AlertTrackerNotification) | uint(lt.AlertStatusNotification)

	sessionHandle.SetAlertMask(alertMask)

	settings := sessionHandle.Settings()
	settings.SetRequestTimeout(config.RequestTimeout)
	settings.SetPeerConnectTimeout(config.PeerConnectTimeout)
	settings.SetAnnounceToAllTrackers(true)
	settings.SetAnnounceToAllTiers(true)
	settings.SetTorrentConnectBoost(config.TorrentConnectBoost)
	settings.SetConnectionSpeed(config.ConnectionSpeed)
	settings.SetMinReconnectTime(config.MinReconnectTime)
	settings.SetMaxFailcount(config.MaxFailCount)
	settings.SetRecvSocketBufferSize(1024 * 1024)
	settings.SetSendSocketBufferSize(1024 * 1024)
	settings.SetRateLimitIpOverhead(true)
	settings.SetMinAnnounceInterval(60)
	settings.SetTrackerBackoff(0)
	settings.SetDiskIoReadMode(0)
	settings.SetDiskIoWriteMode(0)
	sessionHandle.SetSettings(settings)

	err := lt.NewErrorCode()
	rand.Seed(time.Now().UnixNano())
	portLower := config.ListenPort
	if config.RandomPort {
		portLower = rand.Intn(16374) + 49152
	}
	portUpper := portLower + 10
	sessionHandle.ListenOn(lt.NewStd_pair_int_int(portLower, portUpper), err)
	if err.Value() != 0 {
		log.Printf("Error startSession: %v", err.Message())
	}

	settings = sessionHandle.Settings()
	if config.UserAgent != "" {
		settings.SetUserAgent(config.UserAgent)
	} else {
		settings.SetUserAgent(fmt.Sprintf("%s-%s", LibName, LibVersion))
	}
	if config.MaxDownloadRate >= 0 {
		settings.SetDownloadRateLimit(config.MaxDownloadRate * 1024)
	}
	if config.MaxUploadRate >= 0 {
		settings.SetUploadRateLimit(config.MaxUploadRate * 1024)
	}
	settings.SetEnableIncomingTcp(true)
	settings.SetEnableOutgoingTcp(true)
	settings.SetEnableIncomingUtp(true)
	settings.SetEnableOutgoingUtp(true)
	sessionHandle.SetSettings(settings)

	if config.DhtRouters != "" {
		routers := strings.Split(config.DhtRouters, ",")
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
						log.Printf("Error startSession: %v", err)
					}
				}
				sessionHandle.AddDhtRouter(lt.NewStd_pair_string_int(host, port))
				if config.Verbose {
					log.Printf("Added DHT router: %s:%d", host, port)
				}
			}
		}
	}

	if config.Verbose {
		log.Println("Setting encryption settings")
	}
	encryptionSettings := lt.NewPeSettings()
	encryptionSettings.SetOutEncPolicy(byte(lt.LibtorrentPe_settingsEnc_policy(config.Encryption)))
	encryptionSettings.SetInEncPolicy(byte(lt.LibtorrentPe_settingsEnc_policy(config.Encryption)))
	encryptionSettings.SetAllowedEncLevel(byte(lt.PeSettingsBoth))
	encryptionSettings.SetPreferRc4(true)
	sessionHandle.SetPeSettings(encryptionSettings)
}

func startServices() {
	if config.Verbose {
		log.Println("Starting DHT...")
	}
	sessionHandle.StartDht()

	if config.Verbose {
		log.Println("Starting LSD...")
	}
	sessionHandle.StartLsd()

	if config.Verbose {
		log.Println("Starting UPNP...")
	}
	sessionHandle.StartUpnp()

	if config.Verbose {
		log.Println("Starting NATPMP...")
	}
	sessionHandle.StartNatpmp()
}

func stopServices() {
	if config.Verbose {
		log.Println("Stopping DHT...")
	}
	sessionHandle.StopDht()

	if config.Verbose {
		log.Println("Stopping LSD...")
	}
	sessionHandle.StopLsd()

	if config.Verbose {
		log.Println("Stopping UPNP...")
	}
	sessionHandle.StopUpnp()

	if config.Verbose {
		log.Println("Stopping NATPMP...")
	}
	sessionHandle.StopNatpmp()
}

func startHTTP() {
	if config.Verbose {
		log.Println("Starting HTTP Server...")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/ls", lsHandler)
	mux.Handle("/get/", http.StripPrefix("/get/", getHandler(http.FileServer(torrentFs))))
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		forceShutdown <- true
		fmt.Fprintf(w, "OK")
	})
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(torrentFs)))

	handler := http.Handler(mux)

	if config.Verbose {
		log.Printf("Listening HTTP on %s...\n", config.BindAddress)
	}
	s := &http.Server{
		Addr:    config.BindAddress,
		Handler: handler,
	}

	var err error
	httpListener, err = net.Listen("tcp4", config.BindAddress)
	if err != nil {
		log.Printf("Error startHTTP: %v", err)
	} else {
		go s.Serve(httpListener)
	}
}

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, _ := Status()
	w.Write([]byte(status))
}

func lsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	files, _ := Ls()
	w.Write([]byte(files))
}

func getHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index, err := strconv.Atoi(r.URL.String())
		if err == nil && torrentFs.HasTorrentInfo() {
			file, err := torrentFs.FileAt(index)
			if err == nil {
				r.URL.Path = file.Name()
				h.ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})
}

func loop() {
	forceShutdown = make(chan bool, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-forceShutdown:
			httpListener.Close()
			return
		case <-signalChan:
			forceShutdown <- true
		case <-time.After(500 * time.Millisecond):
			consumeAlerts()
			torrentFs.LoadFileProgress()
			if os.Getppid() == 1 {
				forceShutdown <- true
			}
		}
	}
}

func Startup(cfg string) {
	err := json.Unmarshal([]byte(cfg), &config)
	if err != nil {
		log.Printf("Error SetConfig: %s\n", err.Error())
	}

	startSession()
	startServices()
	addTorrent(buildTorrentParams(config.Uri))
	startHTTP()
	loop()
}

func Shutdown() {
	if config.Verbose {
		log.Println("Stopping torrent2http...")
	}
	stopServices()

	torrentFs.Shutdown()
	if session != nil {
		if torrent != nil {
			removeTorrent()
		}
		if config.Verbose {
			log.Println("Aborting the session")
		}
		lt.DeleteSession(session)
	}
	os.Exit(0)
}

func Stop() {
	forceShutdown <- true
}

func Status() (string, error) {
	var status sessionStatus
	if torrent == nil {
		status = sessionStatus{State: -1}
	} else {
		tstatus := torrent.Status()
		status = sessionStatus{
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

func Ls() (string, error) {
	retFiles := lsInfo{}

	if torrentFs.HasTorrentInfo() {
		for _, file := range torrentFs.Files() {
			url := url.URL{
				Scheme: "http",
				Host:   config.BindAddress,
				Path:   "/files/" + file.Name(),
			}
			fi := fileStatusInfo{
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
