package com.bukanir.android.helpers;

import android.content.Context;
import android.os.Environment;
import android.text.TextUtils;

import java.io.File;

public class Storage {

    public static final String TAG = "Storage";

    public static boolean isFreeSpaceAvailable(String storage, Long size) {
        long freeSpace = new File(storage).getUsableSpace();
        if(freeSpace > size) {
            return true;
        }
        return false;
    }

    private static File getDirectory(String variableName) {
        String path = System.getenv(variableName);
        if (!TextUtils.isEmpty(path)) {
            if (path.contains(":")) {
                for (String _path : path.split(":")) {
                    File file = new File(_path);
                    if (file.exists()) {
                        return file;
                    }
                }
            } else {
                File file = new File(path);
                if (file.exists()) {
                    return file;
                }
            }
        }
        return null;
    }

    public static String getStorage(Context context) {
        File externalStorage = null;
        File removableStorage = getDirectory("SECONDARY_STORAGE");

        String state = Environment.getExternalStorageState();
        if(Environment.MEDIA_MOUNTED.equals(state)) {
            externalStorage = Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS);
        }

        if(removableStorage != null && isFreeSpaceAvailable(removableStorage.toString(), (long) 1048576000) && removableStorage.canWrite()) {
            return removableStorage.toString();
        }

        if(externalStorage != null && isFreeSpaceAvailable(externalStorage.toString(), (long) 1048576000) && externalStorage.canWrite()) {
            return externalStorage.toString();
        }

        File hardcodeStorage = new File("/mnt/sdcard");
        if(hardcodeStorage != null && isFreeSpaceAvailable(hardcodeStorage.toString(), (long) 1048576000) && hardcodeStorage.canWrite()) {
            return hardcodeStorage.toString();
        }

        return context.getCacheDir().toString();
    }

}
