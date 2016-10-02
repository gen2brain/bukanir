package com.bukanir.android.application;

import android.content.Context;
import android.content.SharedPreferences;
import android.preference.PreferenceManager;

import com.bukanir.android.entities.Movie;
import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.util.ArrayList;
import java.util.Collections;

public class Favorites {

    private Context context;
    private SharedPreferences preferences;

    public Favorites(Context context) {
        this.context = context.getApplicationContext();
        preferences = PreferenceManager.getDefaultSharedPreferences(this.context);
    }

    public ArrayList<Movie> getFavorites() {
        ArrayList<Movie> favorites = new ArrayList<>();
        String favs = preferences.getString("favorites", "");

        Type listType = new TypeToken<ArrayList<Movie>>() {}.getType();
        ArrayList<Movie> list = new Gson().fromJson(favs, listType);

        if(list != null) {
            favorites = list;
        }
        Collections.sort(favorites);
        return favorites;
    }

    public void addToFavorites(Movie movie) {
        ArrayList<Movie> favorites = getFavorites();
        if(!favorites.contains(movie)) {
            favorites.add(movie);
        }

        String favs = new Gson().toJson(favorites);
        SharedPreferences.Editor editor = preferences.edit();
        editor.putString("favorites", favs);
        editor.apply();
    }

    public void removeFromFavorites(Movie movie) {
        ArrayList<Movie> favorites = getFavorites();
        favorites.remove(movie);

        String favs = new Gson().toJson(favorites);
        SharedPreferences.Editor editor = preferences.edit();
        editor.putString("favorites", favs);
        editor.apply();
    }

}
