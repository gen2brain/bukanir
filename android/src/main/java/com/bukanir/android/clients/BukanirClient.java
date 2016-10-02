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
            Bukanir.SetVerbose(true);
        }
    }

    public static ArrayList<Movie> getTopResults(int category, int limit, int refresh, String cacheDir, int cacheDays) {
        String result = null;
        try {
            result = Bukanir.Category(category, limit, refresh, cacheDir, cacheDays);
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

    public static ArrayList<Movie> getSearchResults(String query, int limit, int refresh, String cacheDir, int cacheDays) {
        String result = null;
        try {
            result = Bukanir.Search(query, limit, refresh, cacheDir, cacheDays);
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

    public static Summary getSummary(int id, int category, int season, int episode) {
        String result = null;
        try {
            result = Bukanir.Summary(id, category, season, episode);
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

    public static ArrayList<Subtitle> getSubtitles(String movie, String year, String release, String language,
        String category, String season, String episode, String imdbId) {
        String result = null;
        try {
            result = Bukanir.Subtitle(movie, year, release, language, Integer.valueOf(category), Integer.valueOf(season), Integer.valueOf(episode), imdbId);
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

    public static ArrayList<AutoComplete> getAutoComplete(String query, int limit) {
        String result = null;
        try {
            result = Bukanir.AutoComplete(query, limit);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null) {
            return null;
        }

        try {
            Type listType = new TypeToken<ArrayList<AutoComplete>>() {}.getType();
            ArrayList<AutoComplete> list = new Gson().fromJson(result, listType);
            return list;
        } catch(Exception e) {
            e.printStackTrace();
            return null;
        }
    }

    public static String getTrailer(String videoId) {
        String result = null;
        try {
            result = Bukanir.Trailer(videoId);
        } catch(Exception e) {
            e.printStackTrace();
        }

        if(result == null) {
            return null;
        }

        return result;
    }

    public static String unzipSubtitle(String url, String dest) {
        String result = null;
        try {
            result = Bukanir.UnzipSubtitle(url, dest);
        } catch(Exception e) {
            e.printStackTrace();
        }

        return result;
    }

}
