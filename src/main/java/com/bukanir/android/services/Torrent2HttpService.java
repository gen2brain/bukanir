package com.bukanir.android.services;

import android.app.Notification;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.IBinder;
import android.preference.PreferenceManager;
import android.support.v4.app.NotificationCompat;
import android.util.Log;
import android.widget.Toast;

import com.bukanir.android.R;
import com.bukanir.android.Torrent2Http;
import com.bukanir.android.activities.MovieActivity;
import com.bukanir.android.utils.Utils;

import java.io.File;
import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.util.ArrayList;
import java.util.Arrays;

public class Torrent2HttpService extends Service {

    public static final String TAG = "Torrent2HttpService";

    String command;
    String cacheDir;
    String encryption;
    String portLower;
    String portUpper;
    String uploadRate;
    String downloadRate;
    File movieDir;
    Process process;
    int id = 313;

    ArrayList<String> trackers = new ArrayList<>(Arrays.asList(
            "udp://tracker.publicbt.com:80/announce",
            "udp://tracker.openbittorrent.com:80/announce",
            "udp://open.demonii.com:1337/announce",
            "udp://tracker.istole.it:6969",
            "udp://tracker.coppersurfer.tk:80"
    ));

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onCreate() {
        command = getApplicationInfo().nativeLibraryDir + "/libtorrent2http.so";

        cacheDir = getExternalCacheDir().toString();
        movieDir = new File(cacheDir + File.separator + "movie");
        movieDir.mkdirs();

        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
        encryption = prefs.getBoolean("encryption", true) ? "1" : "2";
        portLower = String.valueOf(prefs.getInt("port_lower", 6900));
        portUpper = String.valueOf(prefs.getInt("port_upper", 6999));
        uploadRate = prefs.getString("upload_rate", "0");
        downloadRate = prefs.getString("download_rate", "0");

    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        (new Thread() { public void run() {
            try {
                Torrent2Http.shutdown();
                Thread.sleep(2000);
                if(process != null) {
                    process.destroy();
                }
            } catch (Exception e) {
                e.printStackTrace();
            }

            if(movieDir != null) {
                if(movieDir.exists()) {
                    Utils.deleteDir(movieDir);
                }
            }
        }}).start();
        Toast.makeText(this, getString(R.string.torrent_stopped), Toast.LENGTH_LONG).show();
    }

    @Override
    public void onLowMemory() {
        Log.d(TAG, "onLowMemory");
        super.onLowMemory();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        Log.d(TAG, "onStartCommand");

        String magnet = intent.getExtras().getString("magnet");
        String magnetLink = boostMagnet(magnet);

        Torrent2HttpThread thread = new Torrent2HttpThread(this, magnetLink);
        thread.start();

        Toast.makeText(this, getString(R.string.torrent_started), Toast.LENGTH_LONG).show();

        return START_NOT_STICKY;
    }

    private String boostMagnet(String magnet) {
        for(String tracker : trackers){
            String tr = "";
            try {
                tr = URLEncoder.encode(tracker, "UTF-8");
            } catch (UnsupportedEncodingException e) {
                e.printStackTrace();
            }
            magnet += "&tr=" + tr;
        }
        return magnet;
    }

    private class Torrent2HttpThread extends Thread {

        Context context;
        String magnetLink;

        public Torrent2HttpThread(Context ctx, String magnet) {
            context = ctx;
            magnetLink = magnet;
        }

        @Override
        public void run() {
            super.run();
            try {
                ArrayList<String> params = new ArrayList<String>();
                params.add(command);
                params.add("-dlpath");
                params.add(movieDir.toString());
                params.add("-uri");
                params.add(magnetLink);
                params.add("-no-sparse");
                params.add("true");
                params.add("-port-lower");
                params.add(portLower);
                params.add("-port-upper");
                params.add(portUpper);
                params.add("-dlrate");
                params.add(downloadRate);
                params.add("-ulrate");
                params.add(uploadRate);
                params.add("-encryption");
                params.add(encryption);

                ProcessBuilder pb = new ProcessBuilder(params);

                Log.d(TAG, "command:" + pb.command().toString());

                process = pb.start();

                startNotification();

            } catch(Exception e){
                e.getMessage();
            }
        }

        public void startNotification() {
            Intent i = new Intent(context, MovieActivity.class);
            i.setFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_SINGLE_TOP);
            PendingIntent pendIntent = PendingIntent.getActivity(context, 0, i, 0);

            NotificationCompat.Builder builder = new NotificationCompat.Builder(context);
            builder.setTicker(getString(R.string.torrent_started)).setContentTitle(getString(R.string.app_name))
                    .setWhen(System.currentTimeMillis()).setAutoCancel(false)
                    .setOngoing(true).setPriority(Notification.PRIORITY_HIGH)
                    .setContentIntent(pendIntent);
            Notification notification = builder.build();

            notification.flags |= Notification.FLAG_NO_CLEAR;
            startForeground(id, notification);
        }
    }

}
