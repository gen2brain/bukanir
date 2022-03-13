package com.bukanir.android.clients;

import com.bukanir.android.BuildConfig;
import com.bukanir.android.entities.AutoComplete;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.entities.Subtitle;
import com.bukanir.android.entities.Summary;
import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;
import java.lang.reflect.Type;
import java.util.ArrayList;

import go.bukanir.Bukanir;

public class BukanirClient {

    static {
        if(BuildConfig.DEBUG) {
            Bukanir.setVerbose(true);
        }
    }

    public static ArrayList<Movie> getTopResults(int category, int limit, int refresh, String cacheDir, int cacheDays, String tpbHost) {
        String result = null;
        try {
            result = Bukanir.category(category, limit, refresh, cacheDir, cacheDays, tpbHost);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null || result.equals("empty")) {
            return null;
        }

        try {
            Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
            return new Gson().fromJson(result, listType);
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static ArrayList<Movie> getSearchResults(String query, int limit, int refresh, String cacheDir, int cacheDays, int pages, String tpbHost, String eztvHost) {
        String result = null;
        try {
            result = Bukanir.search(query, limit, refresh, cacheDir, cacheDays, pages, tpbHost, eztvHost, "all");
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null || result.equals("empty")) {
            return null;
        }

        try {
            Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
            return new Gson().fromJson(result, listType);
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static Summary getSummary(int id, int category, int season, int episode) {
        String result = null;
        try {
            result = Bukanir.summary(id, category, season, episode);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null || result.equals("empty")) {
            return null;
        }

        try {
            return new Gson().fromJson(result, Summary.class);
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static ArrayList<Subtitle> getSubtitles(String movie, String year, String release, String language,
        String category, String season, String episode, String imdbId) {
        String result = null;
        try {
            result = Bukanir.subtitle(movie, year, release, language, Integer.valueOf(category), Integer.valueOf(season), Integer.valueOf(episode), imdbId);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null || result.equals("empty")) {
            return null;
        }

        try {
            Type listType = new TypeToken<ArrayList<Subtitle>>() {}.getType();
            return new Gson().fromJson(result, listType);
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static ArrayList<AutoComplete> getAutoComplete(String query, int limit) {
        String result = null;
        try {
            result = Bukanir.autoComplete(query, limit);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null || result.equals("empty")) {
            return null;
        }

        try {
            Type listType = new TypeToken<ArrayList<AutoComplete>>() {}.getType();
            return new Gson().fromJson(result, listType);
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static String getTrailer(String videoId) {
        String result = null;
        try {
            result = Bukanir.trailer(videoId);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null || result.equals("empty")) {
            return null;
        }

        return result;
    }

    public static String unzipSubtitle(String url, String dest) {
        String result = null;
        try {
            result = Bukanir.unzipSubtitle(url, dest);
        } catch(Exception e) {
            e.printStackTrace();
        }

        return result;
    }

    public static void cancel() {
        Bukanir.cancel();
    }

}
