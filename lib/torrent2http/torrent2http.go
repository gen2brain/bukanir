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
	Verbose             bool   `json:"verbose"`
}

const (
	LibName    = "libtorrent-rasterbar"
	LibVersion = lt.LIBTORRENT_VERSION
)

var (
	config        Config
	session       lt.Session
	torrent       lt.TorrentHandle
	torrentFs     *torrentFS
	forceShutdown chan bool
	httpListener  net.Listener
)

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

func popAlert() lt.Alert {
	alert := session.PopAlert()
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
		lt.DeleteAlert(alert)
	}
}

func buildTorrentParams(uri string) lt.AddTorrentParams {
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
		if config.Verbose {
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
		if config.Verbose {
			log.Printf("T2HTTP: Will fetch: %s", uri)
		}
		torrentParams.SetUrl(uri)
	}

	if config.Verbose {
		log.Printf("T2HTTP: Setting save path: %s", config.DownloadPath)
	}
	torrentParams.SetSavePath(config.DownloadPath)

	if config.NoSparseFile {
		if config.Verbose {
			log.Println("T2HTTP: Disabling sparse file support...")
		}
		torrentParams.SetStorageMode(lt.StorageModeCompact)
	}

	return torrentParams
}

func addTorrent(torrentParams lt.AddTorrentParams) {
	if config.Verbose {
		log.Println("T2HTTP: Adding torrent")
	}

	error := lt.NewErrorCode()
	defer lt.DeleteErrorCode(error)

	torrent = session.AddTorrent(torrentParams, error)
	if error.Value() != 0 {
		log.Printf("ERROR: addTorrent: %v", error.Message())
	}

	defer lt.DeleteAddTorrentParams(torrentParams)

	if config.Verbose {
		log.Println("T2HTTP: Enabling sequential download")
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
				log.Printf("T2HTTP: Adding tracker: %s", tracker)
			}
			torrent.AddTracker(announceEntry)
			lt.DeleteAnnounceEntry(announceEntry)
		}
	}

	if config.Verbose {
		log.Printf("T2HTTP: Downloading torrent: %s", torrent.Status().GetName())
	}
	torrentFs = newTorrentFS(torrent, config.FileIndex)
}

func removeTorrent() {
	var flag int
	state := torrent.Status().GetState()
	if state != stateCheckingFiles && state != stateQueuedForChecking && !config.KeepFiles {
		flag = int(lt.SessionDeleteFiles)
	}

	if config.Verbose {
		log.Println("T2HTTP: Removing the torrent")
	}
	session.RemoveTorrent(torrent, flag)
}

func startSession() {
	if config.Verbose {
		log.Println("T2HTTP: Starting session...")
		log.Println(fmt.Sprintf("T2HTTP: Library %s-%s", LibName, LibVersion))
	}

	session = lt.NewSession(
		lt.NewFingerprint("LT", lt.LIBTORRENT_VERSION_MAJOR, lt.LIBTORRENT_VERSION_MINOR, 0, 0),
		int(lt.SessionAddDefaultPlugins),
	)

	alertMask := uint(lt.AlertErrorNotification) | uint(lt.AlertStorageNotification) |
		uint(lt.AlertTrackerNotification) | uint(lt.AlertStatusNotification)

	session.SetAlertMask(alertMask)

	settings := session.Settings()

	settings.SetStrictEndGameMode(true)
	settings.SetAnnounceToAllTrackers(true)
	settings.SetAnnounceToAllTiers(true)
	settings.SetAnnounceDoubleNat(true)

	settings.SetRequestTimeout(config.RequestTimeout)
	settings.SetPeerConnectTimeout(config.PeerConnectTimeout)
	settings.SetTorrentConnectBoost(config.TorrentConnectBoost)
	settings.SetConnectionSpeed(config.ConnectionSpeed)
	settings.SetMinReconnectTime(config.MinReconnectTime)
	settings.SetMaxFailcount(config.MaxFailCount)

	settings.SetRateLimitIpOverhead(true)
	settings.SetNoAtimeStorage(true)
	settings.SetPrioritizePartialPieces(false)
	settings.SetFreeTorrentHashes(true)
	settings.SetUseParoleMode(true)
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
	settings.SetIgnoreLimitsOnLocalNetwork(true)
	settings.SetRateLimitUtp(true)
	settings.SetMixedModeAlgorithm(int(lt.SessionSettingsPreferTcp))

	session.SetSettings(settings)

	err := lt.NewErrorCode()
	defer lt.DeleteErrorCode(err)

	rand.Seed(time.Now().UnixNano())
	portLower := config.ListenPort
	if config.RandomPort {
		portLower = rand.Intn(6999-6881) + 6881
	}
	portUpper := portLower + 10

	ports := lt.NewStd_pair_int_int(portLower, portUpper)
	defer lt.DeleteStd_pair_int_int(ports)

	session.ListenOn(ports, err)
	if err.Value() != 0 {
		log.Printf("ERROR: startSession: %v", err.Message())
	}

	settings = session.Settings()
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
	session.SetSettings(settings)

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
						log.Printf("ERROR: startSession: %v", err)
					}
				}

				bind := lt.NewStd_pair_string_int(host, port)
				defer lt.DeleteStd_pair_string_int(bind)

				session.AddDhtRouter(bind)
				if config.Verbose {
					log.Printf("T2HTTP: Added DHT router: %s:%d", host, port)
				}
			}
		}
	}

	if config.Verbose {
		log.Println("T2HTTP: Setting encryption settings")
	}

	encryptionSettings := lt.NewPeSettings()
	defer lt.DeletePeSettings(encryptionSettings)

	encryptionSettings.SetOutEncPolicy(byte(lt.LibtorrentPe_settingsEnc_policy(config.Encryption)))
	encryptionSettings.SetInEncPolicy(byte(lt.LibtorrentPe_settingsEnc_policy(config.Encryption)))
	encryptionSettings.SetAllowedEncLevel(byte(lt.PeSettingsBoth))
	encryptionSettings.SetPreferRc4(true)
	session.SetPeSettings(encryptionSettings)
}

func startServices() {
	if config.Verbose {
		log.Println("T2HTTP: Starting DHT...")
	}
	session.StartDht()

	if config.Verbose {
		log.Println("T2HTTP: Starting LSD...")
	}
	session.StartLsd()

	if config.Verbose {
		log.Println("T2HTTP: Starting UPNP...")
	}
	session.StartUpnp()

	if config.Verbose {
		log.Println("T2HTTP: Starting NATPMP...")
	}
	session.StartNatpmp()
}

func stopServices() {
	if config.Verbose {
		log.Println("T2HTTP: Stopping DHT...")
	}
	session.StopDht()

	if config.Verbose {
		log.Println("T2HTTP: Stopping LSD...")
	}
	session.StopLsd()

	if config.Verbose {
		log.Println("T2HTTP: Stopping UPNP...")
	}
	session.StopUpnp()

	if config.Verbose {
		log.Println("T2HTTP: Stopping NATPMP...")
	}
	session.StopNatpmp()
}

func startHTTP() {
	if config.Verbose {
		log.Println("T2HTTP: Starting HTTP Server...")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/ls", lsHandler)
	mux.Handle("/get/", http.StripPrefix("/get/", getHandler(http.FileServer(torrentFs))))
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		Stop()
		fmt.Fprintf(w, "OK")
	})
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(torrentFs)))

	handler := http.Handler(mux)

	if config.Verbose {
		log.Printf("T2HTTP: Listening HTTP on %s...\n", config.BindAddress)
	}
	s := &http.Server{
		Addr:    config.BindAddress,
		Handler: handler,
	}

	var err error
	httpListener, err = net.Listen("tcp4", config.BindAddress)
	if err != nil {
		log.Printf("ERROR: startHTTP: %v", err)
	} else {
		go s.Serve(httpListener)
	}
}

func stopHTTP() {
	if httpListener != nil {
		err := httpListener.Close()
		if err != nil {
			log.Printf("ERROR: stopHTTP: %v", err)
		}
		httpListener = nil
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
			stopHTTP()
			if config.Verbose {
				log.Println("T2HTTP: Exit loop")
			}
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
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
	}

	startSession()
	startServices()
	addTorrent(buildTorrentParams(config.Uri))
	startHTTP()
	loop()
}

func Shutdown() {
	if config.Verbose {
		log.Println("T2HTTP: Shutdown torrentFs...")
	}
	torrentFs.Shutdown()

	if session != nil {
		if torrent != nil {
			removeTorrent()
		}
		if config.Verbose {
			log.Println("T2HTTP: Aborting the session")
		}
		lt.DeleteSession(session)
	}
}

func Stop() {
	forceShutdown <- true
	//Shutdown()
}

func Started() bool {
	return httpListener != nil
}

func Status() (string, error) {
	var status SessionStatus
	if torrent == nil {
		status = SessionStatus{State: -1}
	} else {
		tstatus := torrent.Status()
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

func Ls() (string, error) {
	retFiles := LsInfo{}

	if torrentFs.HasTorrentInfo() {
		for _, file := range torrentFs.Files() {
			url := url.URL{
				Scheme: "http",
				Host:   config.BindAddress,
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
