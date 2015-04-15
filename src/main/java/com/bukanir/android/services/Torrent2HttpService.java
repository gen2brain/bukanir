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
import java.util.ArrayList;
import java.util.Map;

public class Torrent2HttpService extends Service {

    public static final String TAG = "Torrent2HttpService";

    String libdir;
    String command;
    String encryption;
    String portLower;
    String portUpper;
    String uploadRate;
    String downloadRate;
    File movieDir;
    Process process;
    int id = 313;

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onCreate() {
        libdir = getApplicationInfo().nativeLibraryDir;
        command = libdir + File.separator + "libtorrent2http.so";

        movieDir = new File(Utils.getStorage(this));
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
                try {
                    if(process != null) {
                        process.exitValue();
                    }
                } catch(IllegalThreadStateException e) {
                    process.destroy();
                }
            } catch (Exception e) {
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

        String magnetLink = intent.getExtras().getString("magnet");

        Torrent2HttpThread thread = new Torrent2HttpThread(this, magnetLink);
        thread.start();

        Toast.makeText(this, getString(R.string.torrent_started), Toast.LENGTH_LONG).show();

        return START_NOT_STICKY;
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
                Map<String, String> env = pb.environment();
                env.put("LD_LIBRARY_PATH", libdir);

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
