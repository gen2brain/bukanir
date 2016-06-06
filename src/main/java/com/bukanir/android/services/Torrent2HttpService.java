package com.bukanir.android.services;

import android.app.Notification;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.os.IBinder;
import android.support.v4.app.NotificationCompat;
import android.util.Log;
import android.widget.Toast;

import com.bukanir.android.BuildConfig;
import com.bukanir.android.R;
import com.bukanir.android.application.Settings;
import com.bukanir.android.clients.Torrent2HttpClient;
import com.bukanir.android.activities.MovieActivity;
import com.bukanir.android.entities.TorrentConfig;
import com.bukanir.android.helpers.Storage;
import com.bukanir.android.helpers.Utils;
import com.google.gson.Gson;

import java.io.File;

import go.bukanir.Bukanir;

public class Torrent2HttpService extends Service {

    public static final String TAG = "Torrent2HttpService";

    int id = 313;

    String magnetLink;

    File movieDir;
    File subtitlesDir;

    private Settings settings;
    private TorrentConfig config;

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onCreate() {
        settings = new Settings(this);
        config = new TorrentConfig();

        String storageDir = Storage.getStorage(this) + File.separator + "bukanir";
        movieDir = new File(storageDir + File.separator + "movies");
        subtitlesDir = new File(storageDir + File.separator + "subtitles");

        movieDir.mkdirs();
        subtitlesDir.mkdirs();
    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        super.onDestroy();

        (new Thread() { public void run() {
            Torrent2HttpClient.shutdown();

            if(!settings.keepFiles()) {
                Log.d(TAG, "Removing files");
                if(movieDir != null && movieDir.exists()) {
                    Utils.deleteDir(movieDir);
                }
                if(subtitlesDir != null && subtitlesDir.exists()) {
                    Utils.deleteDir(subtitlesDir);
                }
            }
        }}).start();

        Toast.makeText(getApplicationContext(), getString(R.string.torrent_stopped), Toast.LENGTH_SHORT).show();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        Log.d(TAG, "onStartCommand");

        magnetLink = intent.getExtras().getString("magnet");

        Torrent2HttpThread thread = new Torrent2HttpThread();
        thread.start();

        startNotification();
        Toast.makeText(this, getString(R.string.torrent_started), Toast.LENGTH_SHORT).show();

        return START_NOT_STICKY;
    }

    public void startNotification() {
        Intent i = new Intent(this, MovieActivity.class);
        i.setFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent pendIntent = PendingIntent.getActivity(this, 0, i, 0);

        NotificationCompat.Builder builder = new NotificationCompat.Builder(this);
        builder.setTicker(getString(R.string.torrent_started)).setContentTitle(getString(R.string.app_name))
                .setWhen(System.currentTimeMillis()).setAutoCancel(false)
                .setOngoing(true).setPriority(Notification.PRIORITY_HIGH)
                .setContentIntent(pendIntent);
        Notification notification = builder.build();

        notification.flags |= Notification.FLAG_NO_CLEAR;
        startForeground(id, notification);
    }

    private class Torrent2HttpThread extends Thread {

        @Override
        public void run() {
            super.run();
            try {
                config.uri = magnetLink;
                config.download_path = movieDir.toString();
                config.listen_port = Integer.valueOf(settings.listenPort());
                config.max_download_rate = Integer.valueOf(settings.downloadRate());
                config.max_upload_rate = Integer.valueOf(settings.uploadRate());
                config.encryption = settings.encryption() ? 1 : 2;
                config.keep_files = true;

                if(!settings.seek()) {
                    config.no_sparse_file = true;
                }

                if(BuildConfig.DEBUG) {
                    config.verbose = true;
                }

                Gson gson = new Gson();
                Bukanir.TorrentStartup(gson.toJson(config));
                Bukanir.TorrentShutdown();
            } catch(Exception e){
                e.getMessage();
            }
        }

    }

}
