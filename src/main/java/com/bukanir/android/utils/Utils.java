package com.bukanir.android.utils;

import android.app.Activity;
import android.app.ActivityManager;
import android.app.AlertDialog;
import android.content.Context;
import android.content.DialogInterface;
import android.os.Environment;
import android.view.LayoutInflater;
import android.view.View;

import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.services.BukanirHttpService;
import com.bukanir.android.services.Torrent2HttpService;
import com.google.android.gms.analytics.GoogleAnalytics;
import com.google.android.gms.analytics.Tracker;
import com.thinkfree.showlicense.License;
import com.thinkfree.showlicense.LicensedProject;

import java.io.BufferedInputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

public class Utils {

    public static boolean isStorageAvailable() {
        return Environment.MEDIA_MOUNTED.equals(Environment.getExternalStorageState());
    }

    public static boolean isFreeSpaceAvailable(Context context, Movie m) {
        long freeSpace;

        String sdcard = getExternalSdcardDirectory();
        if(sdcard != null && !sdcard.equals("")) {
            File dir = new File(sdcard);
            freeSpace = dir.getUsableSpace();
        } else {
            freeSpace = context.getExternalCacheDir().getUsableSpace();
        }

        if(freeSpace > Long.valueOf(m.size)) {
            return true;
        }
        return false;
    }

    public static boolean isTorrentServiceRunning(Context context) {
        ActivityManager manager = (ActivityManager) context.getSystemService(Context.ACTIVITY_SERVICE);
        for(ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
            if(Torrent2HttpService.class.getName().equals(service.service.getClassName())) {
                return true;
            }
        }
        return false;
    }

    public static boolean isHttpServiceRunning(Context context) {
        ActivityManager manager = (ActivityManager) context.getSystemService(Context.ACTIVITY_SERVICE);
        for(ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
            if(BukanirHttpService.class.getName().equals(service.service.getClassName())) {
                return true;
            }
        }
        return false;
    }

    public static boolean isX86() {
        String arch = System.getProperty("os.arch").toLowerCase();
        if(arch.startsWith("x86") || arch.startsWith("i686")) {
            return true;
        }
        return false;
    }

    public static String getExternalSdcardDirectory() {
        FileInputStream fis;
        try {
            fis = new FileInputStream(new File("/etc/vold.fstab"));
        } catch (FileNotFoundException e) {
            return null;
        }

        try {
            byte[] buffer = new byte[4096];
            int n;

            String file = "";
            while((n=fis.read(buffer, 0, 4096))>0) {
                file += new String(buffer, 0, n);
            }
            fis.close();

            String[] rows = file.split("\n");
            for(String row: rows) {
                String trimmedRow = row.trim();
                if(trimmedRow.startsWith("#") || trimmedRow.equals("")) {
                    continue;
                } else if(trimmedRow.equals(Environment.getExternalStorageDirectory().getAbsolutePath())) {
                    continue;
                } else {
                    return trimmedRow.split(" ")[2];
                }
            }
        } catch(IOException e) {
        }
        return null;
    }

    public static String getStorage(Context context) {
        String cacheDir;
        String sdcard = getExternalSdcardDirectory();
        if(sdcard != null && !sdcard.equals("")) {
            cacheDir = sdcard;
        } else {
            cacheDir = context.getExternalCacheDir().toString();
        }
        cacheDir = cacheDir + File.separator + "bukanir";
        return cacheDir;
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
        if(dir != null && dir.isDirectory()) {
            String[] children = dir.list();
            for(int i = 0; i < children.length; i++) {
                boolean success = deleteDir(new File(dir, children[i]));
                if(!success) {
                    return false;
                }
            }
        }
        return dir.delete();
    }

    public static InputStream getURL(String uri) {
        try {
            URL url = new URL(uri);
            HttpURLConnection urlConnection = (HttpURLConnection) url.openConnection();
            return urlConnection.getInputStream();
        } catch(Exception e) {
            //e.printStackTrace();
            return null;
        }
    }

    public static void saveURL(String url, String filename) throws IOException {
        BufferedInputStream in = null;
        FileOutputStream fout = null;
        try {
            in = new BufferedInputStream(new URL(url).openStream());
            fout = new FileOutputStream(filename);

            int count;
            final byte data[] = new byte[1024];
            while((count = in.read(data, 0, 1024)) != -1) {
                fout.write(data, 0, count);
            }
        } finally {
            if(in != null) {
                in.close();
            }
            if(fout != null) {
                fout.close();
            }
        }
    }

    public static String unzipSubtitle(String zip, String path) {
        InputStream is;
        ZipInputStream zis;
        try {
            String filename = null;
            is = new FileInputStream(zip);
            zis = new ZipInputStream(new BufferedInputStream(is));
            ZipEntry ze;
            byte[] buffer = new byte[1024];
            int count;

            while((ze = zis.getNextEntry()) != null) {
                filename = ze.getName();

                if(ze.isDirectory()) {
                    File fmd = new File(path + "/" + filename);
                    fmd.mkdirs();
                    continue;
                }

                if(filename.endsWith(".srt") || filename.endsWith(".sub")) {
                    FileOutputStream fout = new FileOutputStream(path + "/" + filename);
                    while((count = zis.read(buffer)) != -1) {
                        fout.write(buffer, 0, count);
                    }
                    fout.close();
                    zis.closeEntry();
                    break;
                }
                zis.closeEntry();
            }
            zis.close();

            File z = new File(zip);
            z.delete();

            return path + "/" + filename;

        } catch(IOException e) {
            e.printStackTrace();
            return null;
        }

    }

    public static void showAbout(Context ctx) {
        LayoutInflater inflater = (LayoutInflater) ctx.getSystemService(Context.LAYOUT_INFLATER_SERVICE);
        View messageView = inflater.inflate(R.layout.about_dialog, null, false);

        String ver = Update.getCurrentVersion(ctx);
        String title = String.format("%s %s", ctx.getResources().getString(R.string.app_name), ver);

        AlertDialog.Builder builder = new AlertDialog.Builder(ctx);
        builder.setIcon(R.drawable.ic_launcher);
        builder.setTitle(title);
        builder.setView(messageView);
        builder.create();
        builder.show();
    }

    public static void showUpdate(Context ctx) {
        final Context context = ctx;
        AlertDialog.Builder builder = new AlertDialog.Builder(ctx);
        builder.setIcon(R.drawable.ic_launcher);
        builder.setTitle(R.string.update_available);
        builder.setMessage(R.string.update_download);
        builder.setPositiveButton("OK",
                new DialogInterface.OnClickListener() {
                    public void onClick(DialogInterface dialog, int whichButton) {
                        Update.downloadUpdate(context);
                    }
                }
        );
        builder.setNegativeButton("Cancel",
                new DialogInterface.OnClickListener() {
                    public void onClick(DialogInterface dialog, int whichButton) {
                        dialog.dismiss();
                    }
                }
        );
        builder.create();
        builder.show();
    }

    public static Tracker getTracker(Context ctx) {
        Tracker tracker;
        String trackingId = "UA-60883832-1";
        Activity activity = (Activity) ctx;

        GoogleAnalytics analytics = GoogleAnalytics.getInstance(ctx);
        analytics.enableAutoActivityReports(activity.getApplication());
        tracker = analytics.newTracker(trackingId);
        tracker.setAnonymizeIp(true);
        return tracker;
    }

    public static long getUnixTime() {
        return System.currentTimeMillis() / 1000L;
    }

    public static LicensedProject[] projectList = new LicensedProject[] {
            new LicensedProject("app-bits icons", null, "http://app-bits.com/free-icons.html", License.CC_BY_ND_3),
            new LicensedProject("gson", null, "https://code.google.com/p/google-gson/", License.APACHE2),
            new LicensedProject("numberpicker", null, "https://github.com/baynezy/numberpicker", License.APACHE2),
            new LicensedProject("showlicense", null, "https://github.com/behumble/showlicense", License.APACHE2),
            new LicensedProject("torrent2http", null, "https://github.com/steeve/torrent2http", License.GPL3),
            new LicensedProject("universal-image-loader", null, "https://github.com/nostra13/Android-Universal-Image-Loader", License.APACHE2),
            new LicensedProject("vitamio", null, "https://github.com/yixia/VitamioBundle", License.APACHE2),
    };

}
