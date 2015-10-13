package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"torrent2http"
)

func parseFlags() {
	config := torrent2http.Config{}
	flag.StringVar(&config.Uri, "uri", "", "Magnet URI or .torrent file URL")
	flag.StringVar(&config.BindAddress, "bind", "localhost:5001", "Bind address of torrent2http")
	flag.StringVar(&config.DownloadPath, "dl-path", ".", "Download path")
	flag.IntVar(&config.FileIndex, "file-index", -1, "Start downloading file with specified index (or start with largest file index otherwise)")
	flag.BoolVar(&config.KeepComplete, "keep-complete", false, "Keep complete files after exiting")
	flag.BoolVar(&config.KeepIncomplete, "keep-incomplete", false, "Keep incomplete files after exiting")
	flag.BoolVar(&config.KeepFiles, "keep-files", false, "Keep all files after exiting (incl. -keep-complete and -keep-incomplete)")
	flag.StringVar(&config.UserAgent, "user-agent", "", "Set an user agent")
	flag.StringVar(&config.DhtRouters, "dht-routers", "router.bittorrent.com:6881,router.utorrent.com:6881,dht.transmissionbt.com:6881,dht.aelitis.com:6881", "Additional DHT routers (comma-separated host:port pairs)")
	flag.StringVar(&config.Trackers, "trackers", "udp://tracker.publicbt.com:80/announce,udp://tracker.openbittorrent.com:80/announce,udp://open.demonii.com:1337/announce,udp://tracker.istole.it:6969,udp://tracker.coppersurfer.tk:80", "Additional trackers (comma-separated URLs)")
	flag.IntVar(&config.ListenPort, "listen-port", 6881, "Use specified port for incoming connections")
	flag.IntVar(&config.TorrentConnectBoost, "torrent-connect-boost", 100, "The number of peers to try to connect to immediately when the first tracker response is received for a torrent")
	flag.IntVar(&config.ConnectionSpeed, "connection-speed", 100, "The number of peer connection attempts that are made per second")
	flag.IntVar(&config.PeerConnectTimeout, "peer-connect-timeout", 2, "The number of seconds to wait after a connection attempt is initiated to a peer")
	flag.IntVar(&config.RequestTimeout, "request-timeout", 5, "The number of seconds until the current front piece request will time out")
	flag.IntVar(&config.MaxDownloadRate, "dl-rate", -1, "Max download rate (kB/s)")
	flag.IntVar(&config.MaxUploadRate, "ul-rate", -1, "Max upload rate (kB/s)")
	flag.IntVar(&config.Encryption, "encryption", 1, "Encryption: 0=forced 1=enabled (default) 2=disabled")
	flag.IntVar(&config.MinReconnectTime, "min-reconnect-time", 60, "The time to wait between peer connection attempts. If the peer fails, the time is multiplied by fail counter")
	flag.IntVar(&config.MaxFailCount, "max-failcount", 3, "The maximum times we try to connect to a peer before stop connecting again")
	flag.BoolVar(&config.NoSparseFile, "no-sparse", false, "Do not use sparse file allocation")
	flag.BoolVar(&config.RandomPort, "random-port", false, "Use random listen port (49152-65535)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Show verbose output")
	flag.Parse()

	if config.Uri == "" {
		flag.Usage()
		os.Exit(1)
	}

	js, err := json.Marshal(config)
	if err != nil {
		log.Printf("Error setting config: %v\n", err.Error())
		os.Exit(1)
	}

	torrent2http.SetConfig(string(js[:]))
}

func main() {
	parseFlags()

	torrent2http.Startup()
	torrent2http.Shutdown()
}
