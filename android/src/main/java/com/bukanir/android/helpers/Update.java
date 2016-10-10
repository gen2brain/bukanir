package com.bukanir.android.helpers;

import android.app.DownloadManager;
import android.content.Context;
import android.content.SharedPreferences;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.net.Uri;
import android.os.Environment;
import android.preference.PreferenceManager;

import java.net.URL;
import java.util.Locale;

import javax.net.ssl.HttpsURLConnection;


public class Update {

    static String getCurrentVersion(Context ctx) {
        try {
            PackageInfo info = ctx.getPackageManager().getPackageInfo(ctx.getPackageName(), 0);
            return info.versionName;
        } catch (PackageManager.NameNotFoundException e) {
            e.printStackTrace();
            return "";
        }
    }

    private static String getUpdateVersion(Context ctx) {
        String ver = getCurrentVersion(ctx);
        float version = Float.parseFloat(ver) + 0.1f;
        return String.format(Locale.ROOT, "%.1f", version);
    }

    public static boolean checkUpdate(Context ctx) {
        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(ctx);
        long now = Utils.getUnixTime();
        long checked = prefs.getLong("update_checked", 0);
        long diff = now - checked;

        if(diff > 86400) {
            SharedPreferences.Editor edit = prefs.edit();
            edit.putLong("update_checked", now);
            edit.apply();
            return true;
        }
        return false;
    }

    private static String getUpdateUrl(Context ctx) {
        String ver = getUpdateVersion(ctx);
        return String.format("https://bukanir.com/download/bukanir-%s.apk", ver);
    }

    public static boolean updateExists(Context ctx) {
        try {
            String url = getUpdateUrl(ctx);
            HttpsURLConnection conn = (HttpsURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("HEAD");
            return (conn.getResponseCode() == 200);
        } catch(Exception e) {
            //e.printStackTrace();
            return false;
        }
    }

    public static void downloadUpdate(Context ctx) {
        DownloadManager downloadmanager;
        downloadmanager = (DownloadManager) ctx.getSystemService(Context.DOWNLOAD_SERVICE);
        Uri uri = Uri.parse(getUpdateUrl(ctx));
        DownloadManager.Request request = new DownloadManager.Request(uri);
        request.setTitle("Downloading update");
        request.setDescription("Bukanir");
        request.setMimeType("application/vnd.android.package-archive");
        request.setVisibleInDownloadsUi(true);
        request.setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, "bukanir-"+getUpdateVersion(ctx)+".apk");
        downloadmanager.enqueue(request);
    }

}
