package com.bukanir.android.activities;

import android.app.SearchManager;
import android.content.Context;
import android.content.Intent;
import android.os.AsyncTask;
import android.os.Build;
import android.support.v4.app.NavUtils;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.ActionBarActivity;
import android.support.v7.app.ActionBar;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.Window;
import android.widget.Toast;

import com.bukanir.android.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.SearchFragment;
import com.bukanir.android.utils.Utils;
import com.thinkfree.showlicense.android.ShowLicense;

import java.util.ArrayList;

public class SearchActivity extends ActionBarActivity {

    public static final String TAG = "SearchActivity";

    private boolean twoPane;
    private ArrayList<Movie> movies;
    private SearchTask searchTask;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        supportRequestWindowFeature(Window.FEATURE_INDETERMINATE_PROGRESS);

        setContentView(R.layout.activity_movie_list);

        if(findViewById(R.id.movie_container) != null) {
            twoPane = true;
        }

        if(savedInstanceState != null) {
            movies = (ArrayList<Movie>) savedInstanceState.getSerializable("search");
        } else {
            Bundle bundle = getIntent().getExtras();
            movies = (ArrayList<Movie>) bundle.get("search");
        }

        final ActionBar actionBar = getSupportActionBar();
        actionBar.setDisplayShowTitleEnabled(true);
        actionBar.setDisplayHomeAsUpEnabled(true);

        beginTransaction(movies);
    }

    @Override
    public void onBackPressed() {
        NavUtils.navigateUpFromSameTask(this);
    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        super.onDestroy();
        cancelSearchTask();
    }

    @Override
    protected void onNewIntent(Intent intent) {
        Log.d(TAG, "onNewIntent");
        setIntent(intent);
        handleSearchIntent(intent);
    }

    @Override
    public void onSaveInstanceState(Bundle outState) {
        Log.d(TAG, "onSaveInstanceState");
        outState.putSerializable("search", movies);
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.search, menu);

        MenuItem searchItem = menu.findItem(R.id.action_search);
        SearchManager searchManager = (SearchManager) getSystemService(Context.SEARCH_SERVICE);
        SearchView searchView = (SearchView) MenuItemCompat.getActionView(searchItem);
        searchView.setSearchableInfo(searchManager.getSearchableInfo(getComponentName()));
        searchView.setIconifiedByDefault(false);

        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();
        switch(id) {
            case R.id.action_settings:
                Intent intent = new Intent(this, SettingsActivity.class);
                startActivity(intent);
                return true;
            case R.id.action_search:
                onSearchRequested();
                return true;
            case android.R.id.home:
                NavUtils.navigateUpFromSameTask(this);
                return true;
            case R.id.action_licenses:
                Intent licenses = ShowLicense.createActivityIntent(this, null, Utils.projectList);
                startActivity(licenses);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    private void beginTransaction(ArrayList<Movie> results) {
        if(twoPane) {
            getSupportFragmentManager().beginTransaction()
                    .replace(R.id.list_container, SearchFragment.newInstance(results, twoPane))
                    .commit();
        } else {
            getSupportFragmentManager().beginTransaction()
                    .replace(R.id.container, SearchFragment.newInstance(results, twoPane))
                    .commit();
        }
    }

    public void cancelSearchTask() {
        if(searchTask != null) {
            if(searchTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                searchTask.cancel(true);
            }
        }
    }

    private void handleSearchIntent(Intent intent) {
        Log.d(TAG, "handleSearchIntent");
        if(Intent.ACTION_SEARCH.equals(intent.getAction())) {
            String query = intent.getStringExtra(SearchManager.QUERY);
            if(Utils.isNetworkAvailable(this)) {
                searchTask = new SearchTask();
                if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                    searchTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, query);
                } else {
                    searchTask.execute(query);
                }
            } else {
                Toast.makeText(this, getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
            }
        }
    }

    private class SearchTask extends AsyncTask<String, Integer, ArrayList<Movie>> {

        @Override
        protected void onPreExecute() {
            super.onPreExecute();
            setSupportProgressBarIndeterminateVisibility(true);
        }

        protected ArrayList<Movie> doInBackground(String... params) {
            String query = params[0];

            ArrayList<Movie> results;
            try {
                results = BukanirClient.getSearchMovies(query, -1);
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }

            return results;
        }

        protected void onPostExecute(ArrayList<Movie> results) {
            setSupportProgressBarIndeterminateVisibility(false);
            if(results != null && !results.isEmpty()) {
                movies = results;
                beginTransaction(results);
            }
        }

    }

}
