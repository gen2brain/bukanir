package com.bukanir.android.clients;

import com.google.gson.Gson;
import com.bukanir.android.entities.TorrentFile;
import com.bukanir.android.entities.TorrentFiles;
import com.bukanir.android.entities.TorrentStatus;

import java.util.Collections;

import go.bukanir.Bukanir;

public class Torrent2HttpClient {

    public boolean waitStartup() {
        return Bukanir.torrentWaitStartup();
    }

    public TorrentStatus getStatus() {
        String s = "";
        try {
            s = Bukanir.torrentStatus();
        } catch(Exception e) {
            e.printStackTrace();
        }

        try {
            Gson gson = new Gson();
            return gson.fromJson(s, TorrentStatus.class);
        } catch(Exception e) {
            return null;
        }
    }

    private TorrentFiles getFiles() {
        String s = "";
        try {
            s = Bukanir.torrentFiles();
        } catch(Exception e) {
            e.printStackTrace();
        }
        try {
            Gson gson = new Gson();
            return gson.fromJson(s, TorrentFiles.class);
        } catch(Exception e) {
            return null;
        }
    }

    public TorrentFile getLargestFile() {
        TorrentFiles torrentFiles = getFiles();
        if(torrentFiles != null && torrentFiles.files.size() > 0) {
            Collections.sort(torrentFiles.files);
            return torrentFiles.files.get(0);
        }
        return null;
    }

}
