package com.bukanir.android.activities;

import android.app.SearchManager;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.support.v4.app.FragmentTransaction;
import android.support.v4.app.NavUtils;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;

import com.bukanir.android.application.Favorites;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MoviesListFragment;
import com.bukanir.android.helpers.Dialogs;

import java.util.ArrayList;

public class FavoritesActivity extends AppCompatActivity {

    public static final String TAG = "FavoritesActivity";

    private boolean twoPane;
    private Favorites favorites;

    private ArrayList<Movie> movies;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        favorites = new Favorites(this);
        setContentView(R.layout.activity_favorites);

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        toolbar.setLogo(R.drawable.ic_launcher);
        setSupportActionBar(toolbar);

        if(getSupportActionBar() != null) {
            getSupportActionBar().setDisplayShowTitleEnabled(true);
            getSupportActionBar().setDisplayHomeAsUpEnabled(true);
        }

        twoPane = findViewById(R.id.movie_container) != null;

        movies = favorites.getFavorites();
        beginTransaction(movies);
    }

    @Override
    protected void onResume() {
        Log.d(TAG, "onResume");
        super.onResume();
        if(movies == null || movies.isEmpty()) {
            movies = favorites.getFavorites();
            beginTransaction(movies);
        }
    }

    @Override
    public void onBackPressed() {
        NavUtils.navigateUpFromSameTask(this);
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.favorites, menu);
        final MenuItem searchItem = menu.findItem(R.id.action_search);

        SearchManager searchManager = (SearchManager) getSystemService(Context.SEARCH_SERVICE);
        SearchView searchView = (SearchView) MenuItemCompat.getActionView(searchItem);
        searchView.setSearchableInfo(searchManager.getSearchableInfo(new ComponentName(getApplicationContext(), SearchActivity.class)));
        searchView.setIconifiedByDefault(false);
        searchView.setSubmitButtonEnabled(true);

        searchView.setOnQueryTextListener(new SearchView.OnQueryTextListener() {
            @Override
            public boolean onQueryTextChange(String newText) {
                return false;
            }

            @Override
            public boolean onQueryTextSubmit(String query) {
                if(searchItem != null) {
                    MenuItemCompat.collapseActionView(searchItem);
                }
                return false;
            }
        });

        searchView.setOnCloseListener(new SearchView.OnCloseListener() {
            @Override
            public boolean onClose() {
                return false;
            }
        });

        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();
        switch(id) {
            case R.id.action_settings:
                startActivity(new Intent(this, SettingsActivity.class));
                return true;
            case R.id.action_search:
                onSearchRequested();
                return true;
            case android.R.id.home:
                NavUtils.navigateUpFromSameTask(this);
                return true;
            case R.id.action_about:
                Dialogs.showAbout(this);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    private void beginTransaction(ArrayList<Movie> results) {
        FragmentTransaction ft;
        if(twoPane) {
            ft = getSupportFragmentManager().beginTransaction();
            ft.replace(R.id.list_container, MoviesListFragment.newInstance(results, twoPane));
            ft.commitAllowingStateLoss();
        } else {
            ft = getSupportFragmentManager().beginTransaction();
            ft.replace(R.id.container, MoviesListFragment.newInstance(results, twoPane));
            ft.commitAllowingStateLoss();
        }
    }

}
