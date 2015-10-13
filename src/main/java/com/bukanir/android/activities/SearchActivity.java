package com.bukanir.android.activities;

import android.app.SearchManager;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.os.AsyncTask;
import android.os.Build;
import android.support.v4.app.FragmentTransaction;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.ProgressBar;
import android.widget.Toast;

import com.bukanir.android.application.Settings;
import com.bukanir.android.clients.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MoviesListFragment;
import com.bukanir.android.helpers.Connectivity;
import com.bukanir.android.helpers.Utils;
import com.google.android.gms.analytics.HitBuilders;
import com.google.android.gms.analytics.Tracker;

import java.util.ArrayList;

public class SearchActivity extends AppCompatActivity {

    public static final String TAG = "SearchActivity";

    private boolean twoPane;
    private Settings settings;

    private SearchTask searchTask;
    private ProgressBar progressBar;

    private String query;
    private String cacheDir;

    private ArrayList<Movie> movies;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        settings = new Settings(this);

        cacheDir = getCacheDir().toString();

        setContentView(R.layout.activity_search);

        progressBar = (ProgressBar) findViewById(R.id.progressbar);

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        toolbar.setLogo(R.drawable.ic_launcher);
        setSupportActionBar(toolbar);

        getSupportActionBar().setDisplayShowTitleEnabled(true);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        if(findViewById(R.id.movie_container) != null) {
            twoPane = true;
        } else {
            twoPane = false;
        }

        if(savedInstanceState != null) {
            query = savedInstanceState.getString("query");
            if(query != null) {
                getSupportActionBar().setTitle(query);
            }

            movies = savedInstanceState.getParcelableArrayList("search");
            if(movies != null) {
                beginTransaction(movies);
            }
        } else {
            handleSearchIntent(getIntent());
        }
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
        if(movies != null && !movies.isEmpty()) {
            outState.putParcelableArrayList("search", movies);
        }
        if(query != null && !query.isEmpty()) {
           outState.putString("query", query);
        }
        super.onSaveInstanceState(outState);
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.main, menu);
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
            case R.id.action_sync:
                if(query != null) {
                    startSearchTask(true);
                }
                return true;
            case android.R.id.home:
                onBackPressed();
                return true;
            case R.id.action_about:
                Utils.showAbout(this);
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

    private void startSearchTask(boolean refresh) {
        Tracker tracker = Utils.getTracker(this);
        tracker.setScreenName(query);
        tracker.send(new HitBuilders.AppViewBuilder().build());

        String force;
        if(refresh) {
            force = "1";
        } else {
            force = "0";
        }

        searchTask = new SearchTask();
        if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
            searchTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, query, force);
        } else {
            searchTask.execute(query, force);
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
            query = intent.getStringExtra(SearchManager.QUERY);
        } else if(Intent.ACTION_VIEW.equals(intent.getAction())) {
            Bundle bundle = intent.getExtras();
            query = bundle.getString("intent_extra_data_key");
        }

        getSupportActionBar().setTitle(query);
        startSearchTask(false);
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

            String force;
            if(params.length == 2) {
                force = params[1];
            } else {
                force = "0";
            }
            int refresh = Integer.parseInt(force);

            ArrayList<Movie> results;

            try {
                results = BukanirClient.getSearchResults(query, -1, refresh, cacheDir, settings.cacheDays());
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }

            if(isCancelled()) {
                return null;
            }

            return results;
        }

        protected void onPostExecute(ArrayList<Movie> results) {
            if(progressBar != null) {
                progressBar.setVisibility(View.GONE);
            }
            if(results != null && !results.isEmpty()) {
                movies = results;
                beginTransaction(results);
            } else {
                if(Connectivity.isConnected(getApplicationContext())) {
                    Toast.makeText(getApplicationContext(), getString(R.string.error_text_connection), Toast.LENGTH_SHORT).show();
                } else {
                    Toast.makeText(getApplicationContext(), getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
                }
            }
        }

    }

}
