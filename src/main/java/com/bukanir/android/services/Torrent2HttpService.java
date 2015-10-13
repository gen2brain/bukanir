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
import com.bukanir.android.helpers.Storage;
import com.bukanir.android.helpers.Utils;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.ArrayList;

public class Torrent2HttpService extends Service {

    public static final String TAG = "Torrent2HttpService";

    int id = 313;

    String binary;
    String magnetLink;

    File movieDir;
    File subtitlesDir;

    private Settings settings;
    private Torrent2HttpThread thread;

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onCreate() {
        settings = new Settings(this);
        binary = getApplicationInfo().nativeLibraryDir + File.separator + "libtorrent2http.so";

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

            if(thread != null) {
                thread.kill();
                thread = null;
            }

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

        thread = new Torrent2HttpThread();
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

        private Process process;

        @Override
        public void run() {
            super.run();
            try {
                String encryption = settings.encryption() ? "1" : "2";

                ArrayList<String> params = new ArrayList<String>();
                params.add(binary);
                params.add("-dl-path");
                params.add(movieDir.toString());
                params.add("-uri");
                params.add(magnetLink);
                params.add("-listen-port");
                params.add(settings.listenPort());
                params.add("-dl-rate");
                params.add(settings.downloadRate());
                params.add("-ul-rate");
                params.add(settings.uploadRate());
                params.add("-encryption");
                params.add(encryption);
                params.add("-keep-files");

                if(!settings.seek()) {
                    params.add("-no-sparse");
                }

                if(BuildConfig.DEBUG) {
                    params.add("-verbose");
                }

                ProcessBuilder pb = new ProcessBuilder(params);
                Log.d(TAG, android.text.TextUtils.join(" ", pb.command()));

                process = pb.start();

                new StreamGobbler(process.getErrorStream()).start();

            } catch(Exception e){
                e.getMessage();
            }
        }

        public void kill() {
            try {
                if(process != null) {
                    process.exitValue();
                }
            } catch(IllegalThreadStateException e) {
                if(process != null) {
                    process.destroy();
                    process = null;
                }
            }
        }

        private class StreamGobbler extends Thread {
            InputStream is;

            private StreamGobbler(InputStream is) {
                this.is = is;
            }

            @Override
            public void run() {
                try {
                    InputStreamReader isr = new InputStreamReader(is);
                    BufferedReader br = new BufferedReader(isr);
                    String line;
                    while((line = br.readLine()) != null) {
                        Log.d("Torrent2Http", line);
                    }
                } catch(IOException e) {
                    e.printStackTrace();
                }
            }
        }

    }

}
