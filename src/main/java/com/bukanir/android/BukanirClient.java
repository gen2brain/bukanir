package com.bukanir.android;

import com.bukanir.android.entities.Movie;
import com.bukanir.android.entities.Subtitle;
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

import go.main.Main;

public class BukanirClient {

    public static String HOST = "127.0.0.1";
    public static String PORT = "7314";
    public static final String URL = String.format("http://%s:%s/", HOST, PORT);

    public static ArrayList<Movie> getTopMovies(String category, int limit, boolean refresh, String cacheDir) {
        if(Utils.isX86()) {
            String url = String.format("%scategory?c=%s&t=%s", URL, category, cacheDir);
            if(limit != -1) {
                url = url + "&l=" + String.valueOf(limit);
            }
            if(refresh) {
                url = url + "&f=1";
            }
            InputStream input = Utils.getURL(url);
            if (input == null) {
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
        } else {
            int force;
            if(refresh) {
                force = 1;
            } else {
                force = 0;
            }

            String result = null;
            try {
                result = Main.Category(category, limit, force, cacheDir);
            } catch(Exception e) {
                e.printStackTrace();
            }

            if(result == null) {
                return null;
            }

            try {
                Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
                ArrayList<Movie> list = new Gson().fromJson(result, listType);
                return list;
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }
        }
    }

    public static ArrayList<Movie> getSearchMovies(String query, int limit) {
        if(Utils.isX86()) {
            String encodedQuery = query;
            try {
                encodedQuery = URLEncoder.encode(query, "UTF-8");
            } catch (UnsupportedEncodingException e) {
                e.printStackTrace();
            }

            String url = URL + "search?q=" + encodedQuery;
            if (limit != -1) {
                url = url + "&l=" + String.valueOf(limit);
            }

            InputStream input = Utils.getURL(url);
            if (input == null) {
                return null;
            }
            Reader reader = new InputStreamReader(input);
            try {
                Type listType = new TypeToken<ArrayList<Movie>>() {
                }.getType();
                ArrayList<Movie> list = new Gson().fromJson(reader, listType);
                return list;
            } catch (Exception e) {
                e.printStackTrace();
                return null;
            }
        } else {
            String result = null;
            try {
                result = Main.Search(query, limit);
            } catch(Exception e) {
                e.printStackTrace();
            }

            if(result == null) {
                return null;
            }

            try {
                Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
                ArrayList<Movie> list = new Gson().fromJson(result, listType);
                return list;
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }
        }
    }

    public static Summary getSummary(int id, int category, int season) {
        if(Utils.isX86()) {
            String url = URL + "summary?i=" + String.valueOf(id) + "&c=" + String.valueOf(category);

            InputStream input = Utils.getURL(url);
            if (input == null) {
                return null;
            }

            Reader reader = new InputStreamReader(input);
            try {
                Summary summary = new Gson().fromJson(reader, Summary.class);
                return summary;
            } catch (Exception e) {
                e.printStackTrace();
                return null;
            }
        } else {
            String result = null;
            try {
                result = Main.Summary(id, category, season);
            } catch(Exception e) {
                e.printStackTrace();
            }

            if(result == null) {
                return null;
            }

            try {
                Summary summary = new Gson().fromJson(result, Summary.class);
                return summary;
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }
        }
    }

    public static ArrayList<Subtitle> getSubtitles(String movie, String year, String release,String language,
                                                   String category, String season, String episode) {
        if(Utils.isX86()) {
            String encodedMovie = movie;
            String encodedRelease = release;
            try {
                encodedMovie = URLEncoder.encode(movie, "UTF-8");
                encodedRelease = URLEncoder.encode(release, "UTF-8");
            } catch (UnsupportedEncodingException e) {
                e.printStackTrace();
            }

            String url = String.format(URL + "subtitle?m=%s&y=%s&r=%s&l=%s&c=%s&s=%s&e=%s",
                    encodedMovie, year, encodedRelease, language, category, season, episode);

            InputStream input = Utils.getURL(url);
            if (input == null) {
                return null;
            }
            Reader reader = new InputStreamReader(input);
            try {
                Type listType = new TypeToken<ArrayList<Subtitle>>() {
                }.getType();
                ArrayList<Subtitle> list = new Gson().fromJson(reader, listType);
                return list;
            } catch (Exception e) {
                e.printStackTrace();
                return null;
            }
        } else {
            String result = null;
            try {
                result = Main.Subtitle(movie, year, release, language, Integer.valueOf(category), Integer.valueOf(season), Integer.valueOf(episode));
            } catch(Exception e) {
                e.printStackTrace();
            }

            if(result == null) {
                return null;
            }

            try {
                Type listType = new TypeToken<ArrayList<Subtitle>>() {}.getType();
                ArrayList<Subtitle> list = new Gson().fromJson(result, listType);
                return list;
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }
        }
    }

}
