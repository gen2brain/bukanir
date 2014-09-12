package com.bukanir.android;

import com.bukanir.android.entities.Movie;
import com.bukanir.android.entities.Summary;
import com.bukanir.android.utils.Utils;
import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;

import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.Reader;
import java.io.UnsupportedEncodingException;
import java.lang.reflect.Type;
import java.net.URLEncoder;
import java.util.ArrayList;

public class BukanirClient {

    public static String HOST = "127.0.0.1";
    public static String PORT = "7314";
    public static final String URL = String.format("http://%s:%s/", HOST, PORT);

    public static ArrayList<Movie> getTopMovies(String category, int limit, boolean refresh) {
        String url = URL + "category/" + category;
        if(limit != -1) {
            url = url + "/" + String.valueOf(limit);
        }
        if(refresh) {
            url = url + "/1";
        }
        InputStream input = Utils.getURL(url);
        if(input == null) {
            return null;
        }
        Reader reader = new InputStreamReader(input);
        try {
            Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
            ArrayList<Movie> list = new Gson().fromJson(reader, listType);
            return list;
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static ArrayList<Movie> getSearchMovies(String query, int limit) {
        String encodedQuery = query;
        try {
            encodedQuery = URLEncoder.encode(query, "UTF-8");
        } catch (UnsupportedEncodingException e) {
        }

        String url = URL + "search/" + encodedQuery;
        if(limit != -1) {
            url = url + "/" + String.valueOf(limit);
        }

        InputStream input = Utils.getURL(url);
        if(input == null) {
            return null;
        }
        Reader reader = new InputStreamReader(input);
        try {
            Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
            ArrayList<Movie> list = new Gson().fromJson(reader, listType);
            return list;
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static Summary getSummary(int id) {
        String url = URL + "summary/" + String.valueOf(id);

        InputStream input = Utils.getURL(url);
        if(input == null) {
            return null;
        }

        Reader reader = new InputStreamReader(input);
        try {
            Summary summary = new Gson().fromJson(reader, Summary.class);
            return summary;
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

}
