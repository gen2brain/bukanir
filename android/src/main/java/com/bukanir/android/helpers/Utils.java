package com.bukanir.android.helpers;

import android.app.Activity;
import android.app.ActivityManager;
import android.content.Context;

import com.bukanir.android.services.TorrentService;
import com.google.android.gms.analytics.GoogleAnalytics;
import com.google.android.gms.analytics.Tracker;

import java.io.File;

public class Utils {

    public static boolean isTorrentServiceRunning(Context context) {
        ActivityManager manager = (ActivityManager) context.getSystemService(Context.ACTIVITY_SERVICE);
        for(ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
            if(TorrentService.class.getName().equals(service.service.getClassName())) {
                return true;
            }
        }
        return false;
    }

    public static String toTitleCase(String input) {
        StringBuilder titleCase = new StringBuilder();
        boolean nextTitleCase = true;
        for(char c : input.toCharArray()) {
            if(Character.isSpaceChar(c)) {
                nextTitleCase = true;
            } else if(nextTitleCase) {
                c = Character.toTitleCase(c);
                nextTitleCase = false;
            }
            titleCase.append(c);
        }
        return titleCase.toString();
    }

    public static boolean deleteDir(File dir) {
        if (dir != null && dir.isDirectory()) {
            String[] children = dir.list();
            for (String aChildren : children) {
                boolean success = deleteDir(new File(dir, aChildren));
                if (!success) {
                    return false;
                }
            }
        }

        return dir != null && dir.delete();
    }

    public static Tracker getTracker(Context ctx) {
        Tracker tracker;
        String trackingId = "UA-60883832-1";
        Activity activity = (Activity) ctx;

        GoogleAnalytics analytics = GoogleAnalytics.getInstance(ctx);
        analytics.enableAutoActivityReports(activity.getApplication());
        analytics.setLocalDispatchPeriod(600);
        tracker = analytics.newTracker(trackingId);
        tracker.setAnonymizeIp(true);
        return tracker;
    }

    static long getUnixTime() {
        return System.currentTimeMillis() / 1000L;
    }

}
