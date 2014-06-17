package com.bukanir.android;

import com.google.gson.Gson;
import com.bukanir.android.entities.TorrentFile;
import com.bukanir.android.entities.TorrentFiles;
import com.bukanir.android.entities.TorrentStatus;
import com.bukanir.android.utils.Utils;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.Reader;
import java.util.Collections;

public class Torrent2Http {

    public static String HOST = "127.0.0.1";
    public static String PORT = "5001";
    public static final String URL = String.format("http://%s:%s/", HOST, PORT);

    public int T2H_POLL = 1000;
    public long T2H_TIMEOUT = 30000;

    public enum State {
        T2H_QUEUED,
        T2H_CHECKING,
        T2H_DOWNLOADING_METADATA,
        T2H_DOWNLOADING,
        T2H_FINISHED,
        T2H_SEEDING,
        T2H_ALLOCATING,
        T2H_ALLOCATING_FILE;
    };

    public boolean waitStartup() {
        long start = System.currentTimeMillis();
        while((System.currentTimeMillis() - start) < T2H_TIMEOUT) {
            TorrentStatus status = getStatus();
            if(status != null) {
                return true;
            }
            try {
                Thread.sleep(T2H_POLL);
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
        }
        return false;
    }

    public TorrentStatus getStatus() {
        Gson gson = new Gson();
        InputStream input = Utils.getURL(URL + "status");
        if(input == null) {
            return null;
        }
        Reader reader = new InputStreamReader(input);
        try {
            TorrentStatus status = gson.fromJson(reader, TorrentStatus.class);
            return status;
        } catch(Exception e) {
            return null;
        }
    }

    public TorrentFiles getFiles() {
        Gson gson = new Gson();
        InputStream input = Utils.getURL(URL + "ls");
        if(input == null) {
            return null;
        }
        Reader reader = new InputStreamReader(input);
        TorrentFiles files = gson.fromJson(reader, TorrentFiles.class);
        return files;
    }

    public TorrentFile getLargestFile() {
        TorrentFiles torrentFiles = getFiles();
        Collections.sort(torrentFiles.files);
        return torrentFiles.files.get(0);

    }

    public static void shutdown() {
       Utils.getURL(URL + "shutdown");
    }

}
