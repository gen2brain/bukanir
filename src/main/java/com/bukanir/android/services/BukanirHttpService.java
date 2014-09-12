package com.bukanir.android.services;

import android.app.Service;
import android.content.Context;
import android.content.Intent;
import android.os.IBinder;
import android.util.Log;

import java.util.ArrayList;


public class BukanirHttpService extends Service {

    public static final String TAG = "BukanirHttpService";

    String command;
    String cacheDir;
    Process process;

    public static final String host = "127.0.0.1";
    public static final String bind = ":7314";
    public static final String url = String.format("http://%s%s/", host, bind);

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onCreate() {
        command = getApplicationInfo().nativeLibraryDir + "/libbukanir-http.so";
        cacheDir = getCacheDir().toString();
        Log.d(TAG, command);
    }


    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        (new Thread() { public void run() {
            try {
                if(process != null) {
                    process.destroy();
                }
            } catch(Exception e) {
                e.printStackTrace();
            }
        }}).start();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        Log.d(TAG, "onStartCommand");

        BukanirHttpThread thread = new BukanirHttpThread(this);
        thread.start();

        return START_NOT_STICKY;
    }


    private class BukanirHttpThread extends Thread {

        Context context;

        public BukanirHttpThread(Context ctx) {
            context = ctx;
        }

        @Override
        public void run() {
            super.run();
            try {
                ArrayList<String> params = new ArrayList<String>();
                params.add(command);
                params.add("-bind");
                params.add(bind);
                params.add("-cachedir");
                params.add(cacheDir);
                ProcessBuilder pb = new ProcessBuilder(params);
                Log.d(TAG, pb.command().toString());
                process = pb.start();
            } catch(Exception e){
                e.printStackTrace();
            }
        }

    }

}
