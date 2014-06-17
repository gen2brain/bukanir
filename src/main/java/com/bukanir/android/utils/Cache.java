package com.bukanir.android.utils;

import android.content.Context;

import com.bukanir.android.entities.Movie;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.ObjectInputStream;
import java.io.ObjectOutputStream;
import java.util.ArrayList;

public class Cache {

    public static boolean saveObject(String key, File cacheDir, ArrayList<Movie> obj) {
        final File suspend_f=new File(cacheDir, key);

        FileOutputStream fos = null;
        ObjectOutputStream oos = null;
        boolean keep = true;

        try {
            fos = new FileOutputStream(suspend_f);
            oos = new ObjectOutputStream(fos);
            oos.writeObject(obj);
        } catch(Exception e) {
            keep = false;
        } finally {
            try {
                if(oos != null) {
                    oos.close();
                }
                if(fos != null) {
                    fos.close();
                }
                if(keep == false) {
                    suspend_f.delete();
                }
            } catch(Exception e) {
            }
        }

        return keep;
    }

    public static ArrayList<Movie> getObject(String key, File cacheDir, Context context) {
        final File suspend_f=new File(cacheDir, key);

        long mtime = suspend_f.lastModified();
        long ptime = System.currentTimeMillis();
        if((ptime - mtime)/1000 > (86400 * 2)) {
            return null;
        }

        ArrayList<Movie> simpleClass= null;
        FileInputStream fis = null;
        ObjectInputStream is = null;

        try {
            fis = new FileInputStream(suspend_f);
            is = new ObjectInputStream(fis);
            simpleClass = (ArrayList<Movie>) is.readObject();
        } catch(Exception e) {
            String val= e.getMessage();
        } finally {
            try {
                if(fis != null) {
                    fis.close();
                }
                if(is != null) {
                    is.close();
                }
            } catch(Exception e) {
            }
        }

        return simpleClass;
    }

}
