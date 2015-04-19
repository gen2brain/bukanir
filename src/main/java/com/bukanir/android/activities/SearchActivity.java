package com.bukanir.android.activities;

import android.app.AlertDialog;
import android.app.SearchManager;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.os.AsyncTask;
import android.os.Build;
import android.support.v4.app.NavUtils;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.ActionBarActivity;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.ProgressBar;
import android.widget.Toast;

import com.bukanir.android.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.SearchFragment;
import com.bukanir.android.utils.Connectivity;
import com.bukanir.android.utils.Utils;
import com.google.android.gms.analytics.HitBuilders;
import com.google.android.gms.analytics.Tracker;
import com.thinkfree.showlicense.android.ShowLicense;

import java.util.ArrayList;

import go.Go;

public class SearchActivity extends ActionBarActivity {

    public static final String TAG = "SearchActivity";

    private boolean twoPane;
    private ArrayList<Movie> movies;
    private SearchTask searchTask;
    private ProgressBar progressBar;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        Go.init(getApplicationContext());

        setContentView(R.layout.activity_search);

        progressBar = (ProgressBar) findViewById(R.id.progressbar);

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        toolbar.setLogo(R.drawable.ic_launcher);
        setSupportActionBar(toolbar);

        getSupportActionBar().setDisplayShowTitleEnabled(true);

        if(findViewById(R.id.movie_container) != null) {
            twoPane = true;
        }

        if(savedInstanceState != null) {
            movies = (ArrayList<Movie>) savedInstanceState.getSerializable("search");
            if(movies != null) {
                beginTransaction(movies);
            }
        } else {
            handleSearchIntent(getIntent());
        }
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
                if (searchItem != null) {
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
                AlertDialog licenses = ShowLicense.createDialog(this, null, Utils.projectList);
                licenses.setIcon(R.drawable.ic_launcher);
                licenses.setTitle(getString(R.string.action_licenses));
                licenses.show();
                return true;
            case R.id.action_about:
                Utils.showAbout(this);
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
            getSupportActionBar().setSubtitle(query);

            if(Connectivity.isConnected(this)) {
                Tracker tracker = Utils.getTracker(this);
                tracker.setScreenName(query);
                tracker.send(new HitBuilders.AppViewBuilder().build());

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
            if(progressBar != null) {
                progressBar.setVisibility(View.VISIBLE);
            }
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
            if(progressBar != null) {
                progressBar.setVisibility(View.INVISIBLE);
            }
            if(results != null && !results.isEmpty()) {
                movies = results;
                beginTransaction(results);
            }
        }

    }

}
