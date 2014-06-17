package com.bukanir.android.utils;

import android.app.ActivityManager;
import android.content.Context;
import android.net.ConnectivityManager;
import android.net.NetworkInfo;
import android.os.Environment;

import com.bukanir.android.entities.Movie;
import com.bukanir.android.services.Torrent2HttpService;
import com.thinkfree.showlicense.License;
import com.thinkfree.showlicense.LicensedProject;

import net.ricecode.similarity.JaroWinklerStrategy;
import net.ricecode.similarity.SimilarityStrategy;
import net.ricecode.similarity.StringSimilarityService;
import net.ricecode.similarity.StringSimilarityServiceImpl;

import org.apache.http.HttpResponse;
import org.apache.http.HttpStatus;
import org.apache.http.client.HttpClient;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.impl.client.DefaultHttpClient;
import org.apache.http.params.BasicHttpParams;
import org.apache.http.params.HttpConnectionParams;
import org.apache.http.params.HttpParams;

import java.io.BufferedInputStream;
import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.InetAddress;
import java.net.SocketTimeoutException;
import java.net.URI;
import java.net.URL;
import java.text.DecimalFormat;
import java.util.Arrays;
import java.util.List;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

public class Utils {

    public static boolean isNetworkAvailable(Context context) {
        final ConnectivityManager conMgr =  (ConnectivityManager) context.getSystemService(Context.CONNECTIVITY_SERVICE);
        final NetworkInfo activeNetwork = conMgr.getActiveNetworkInfo();
        if (activeNetwork != null && activeNetwork.isConnected()) {
            return true;
        } else {
            return false;
        }
    }

    public static boolean isNetworkReachableInet(String host) {
        try {
            return InetAddress.getByName(host).isReachable(3000);
        } catch(IOException e) {
            return false;
        }
    }

    public static boolean isNetworkReachable(String host) {
        boolean result = false;
        try {
            HttpGet request = new HttpGet("http://" + host.toString());
            HttpParams httpParameters = new BasicHttpParams();
            HttpConnectionParams.setConnectionTimeout(httpParameters, 3000);
            HttpClient httpClient = new DefaultHttpClient(httpParameters);
            HttpResponse response = httpClient.execute(request);

            int status = response.getStatusLine().getStatusCode();
            if (status == HttpStatus.SC_OK) {
                result = true;
            }
        } catch(SocketTimeoutException e) {
            result = false;
        } catch(IOException e) {
            result = false;
        }
        return result;
    }

    public static boolean isStorageAvailable() {
        return Environment.MEDIA_MOUNTED.equals(Environment.getExternalStorageState());
    }

    public static boolean isFreeSpaceAvailable(Context ctx, Movie m) {
        int movieSize = getMovieSize(m);

        String cacheDir = ctx.getExternalCacheDir().toString();
        int freeSpace = (int) (new File(cacheDir).getFreeSpace() / 1048576);

        if(freeSpace > movieSize) {
            return true;
        }
        return false;
    }

    public static boolean isServiceRunning(Context context) {
        ActivityManager manager = (ActivityManager) context.getSystemService(Context.ACTIVITY_SERVICE);
        for(ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
            if(Torrent2HttpService.class.getName().equals(service.service.getClassName())) {
                return true;
            }
        }
        return false;
    }

    public static boolean isStorageVfat(Context context) {
        try {
            String cacheDir = context.getExternalCacheDir().toString();
            List<String> items = Arrays.asList(cacheDir.split("/"));
            String path = items.get(1) + "/" + items.get(2);

            String cmd = String.format("/system/bin/mount | grep '%s'", path);
            String[] command = {"/system/bin/sh", "-c", cmd};

            Process process = Runtime.getRuntime().exec(command, null, new File("/system/bin"));
            try {
                process.waitFor();
            } catch(InterruptedException e) {
                e.printStackTrace();
            }

            String line;
            String output = "";
            BufferedReader in = new BufferedReader(new InputStreamReader(process.getInputStream()));
            while((line = in.readLine()) != null) {
                output += line;
            }

            List<String> outputItems = Arrays.asList(output.split(" "));
            if(outputItems.size() >= 3) {
                if(outputItems.get(2).equals("vfat")) {
                    return true;
                }
            }
        } catch (IOException e) {
            e.printStackTrace();
        }
        return false;
    }

    public static int getMovieSize(Movie m) {
        int movieSize = 0;
        if(!m.size.isEmpty()) {
            try {
                if(m.size.endsWith("GiB")) {
                    movieSize = (int) (Float.parseFloat(m.size.replaceAll(".*?([\\d.]+).*", "$1")) * 1024);
                } else if(m.size.endsWith("MiB")) {
                    movieSize = (int) Float.parseFloat(m.size.replaceAll(".*?([\\d.]+).*", "$1"));
                }
            } catch(NumberFormatException e) {
                e.printStackTrace();
                movieSize = 0;
            }
        }
        return movieSize;
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

    public static InputStream getURL(String url){
        URI uri;
        InputStream data = null;
        DefaultHttpClient httpClient = new DefaultHttpClient();
        try {
            uri = new URI(url);
            HttpGet method = new HttpGet(uri);
            HttpResponse response = httpClient.execute(method);
            data = response.getEntity().getContent();
        } catch(Exception e) {
        }
        return data;
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

    public static String join(List<String> array, String joinChar) {
        String joinedString = "";
        boolean firstFlag = true;
        for(String curString: array) {
            if(firstFlag) {
                joinedString += curString;
                firstFlag = false;
            } else {
                joinedString += joinChar + curString;
            }
        }
        return joinedString;
    }

    public static String compareRelease(String torrentRelease, String subtitleRelease) {
        SimilarityStrategy strategy = new JaroWinklerStrategy();
        StringSimilarityService service = new StringSimilarityServiceImpl(strategy);
        torrentRelease = torrentRelease.replace(".", " ").replace("-", " ");
        subtitleRelease = subtitleRelease.replace(".", " ").replace("-", " ");
        DecimalFormat df = new DecimalFormat("#.##");
        double score = service.score(torrentRelease, subtitleRelease);
        return df.format(score);
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

    public static LicensedProject[] projectList = new LicensedProject[] {
            new LicensedProject("app-bits icons", null, "http://app-bits.com/free-icons.html", License.CC_BY_ND_3),
            new LicensedProject("gson", null, "https://code.google.com/p/google-gson/", License.APACHE2),
            new LicensedProject("jsoup", null, "http://jsoup.org/", License.MIT),
            new LicensedProject("numberpicker", null, "https://github.com/baynezy/numberpicker", License.APACHE2),
            new LicensedProject("retrofit", null, "https://github.com/square/retrofit", License.APACHE2),
            new LicensedProject("showlicense", null, "https://github.com/behumble/showlicense", License.APACHE2),
            new LicensedProject("string-similarity", null, "https://github.com/rrice/java-string-similarity", License.MIT),
            new LicensedProject("tmdb-java", null, "https://github.com/UweTrottmann/tmdb-java", License.APACHE2),
            new LicensedProject("torrent2http", null, "https://github.com/steeve/torrent2http", License.GPL3),
            new LicensedProject("universal-image-loader", null, "https://github.com/nostra13/Android-Universal-Image-Loader", License.APACHE2),
            new LicensedProject("vitamio", null, "https://github.com/yixia/VitamioBundle", License.APACHE2),
    };

}
