package com.bukanir.android.entities;

public class TorrentConfig {

    public String uri;
    public String bind_address;
    public int file_index;
    public int max_upload_rate;
    public int max_download_rate;
    public String download_path;
    public String user_agent;
    public boolean keep_complete;
    public boolean keep_incomplete;
    public boolean keep_files;
    public int encryption;
    public boolean no_sparse_file;
    public int peer_connect_timeout;
    public int request_timeout;
    public int torrent_connect_boost;
    public int connection_speed;
    public int listen_port;
    public int min_reconnect_time;
    public int max_fail_count;
    public boolean random_port;
    public String dht_routers;
    public String trackers;
    public boolean verbose;

    public TorrentConfig() {
        uri = "";
        bind_address = "127.0.0.1:5001";
        file_index = -1;
        max_upload_rate = -1;
        max_download_rate = -1;
        download_path = "";
        user_agent = "";
        keep_complete = false;
        keep_incomplete = false;
        keep_files = false;
        encryption = 1;
        no_sparse_file = false;
        peer_connect_timeout = 2;
        request_timeout = 5;
        torrent_connect_boost = 100;
        connection_speed = 100;
        listen_port = 6881;
        min_reconnect_time = 60;
        max_fail_count = 3;
        random_port = false;
        dht_routers = "router.bittorrent.com:6881,router.utorrent.com:6881,dht.transmissionbt.com:6881,dht.aelitis.com:6881";
        trackers = "udp://tracker.publicbt.com:80/announce,udp://tracker.openbittorrent.com:80/announce,udp://open.demonii.com:1337/announce,udp://tracker.istole.it:6969,udp://tracker.coppersurfer.tk:80";
        verbose = false;
    }

}
