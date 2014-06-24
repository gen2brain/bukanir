package com.bukanir.android.activities;

import android.app.SearchManager;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.AsyncTask;
import android.os.Build;
import android.preference.PreferenceManager;
import android.support.v4.app.Fragment;
import android.support.v4.app.FragmentTransaction;
import android.support.v4.app.NavUtils;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.ActionBarActivity;
import android.support.v7.app.ActionBar;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.widget.Toast;

import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.ProgressFragment;
import com.bukanir.android.fragments.SearchFragment;
import com.bukanir.android.scrapers.TheMovieDb;
import com.bukanir.android.scrapers.ThePirateBay;
import com.bukanir.android.utils.Utils;
import com.thinkfree.showlicense.android.ShowLicense;

import java.net.UnknownHostException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class SearchActivity extends ActionBarActivity {

    public static final String TAG = "SearchActivity";

    private boolean proxy;
    private boolean twoPane;
    private ArrayList<Movie> movies;
    private SearchTask searchTask;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

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
        if (Intent.ACTION_SEARCH.equals(intent.getAction())) {
            String query = intent.getStringExtra(SearchManager.QUERY);
            if(Utils.isNetworkAvailable(this)) {
                SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
                proxy = prefs.getBoolean("proxy", true);

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

        String query;
        ProgressFragment progressFragment;

        @Override
        protected void onPreExecute() {
            super.onPreExecute();
            progressFragment = ProgressFragment.newInstance(getString(R.string.downloading_metadata));
            FragmentTransaction ft = getSupportFragmentManager().beginTransaction();
            if(twoPane) {
                Fragment prev = getSupportFragmentManager().findFragmentByTag("dialog");
                if (prev != null) {
                    ft.remove(prev);
                }
                progressFragment.show(getSupportFragmentManager(), "dialog");
            } else {
                Fragment prev = getSupportFragmentManager().findFragmentById(R.id.container);
                if (prev != null) {
                    ft.remove(prev);
                }
                ft.replace(R.id.container, progressFragment);
                ft.commit();
            }
        }

        protected ArrayList<Movie> doInBackground(String... params) {

            query = params[0];
            ArrayList<Movie> results = new ArrayList<Movie>();

            ThePirateBay tpb = new ThePirateBay(proxy);
            List<ArrayList<String>> torrents = new ArrayList<>();

            if(isCancelled()) {
                return null;
            }

            try {
                if(query != null) {
                    ArrayList<ArrayList<String>> search = tpb.search(query, ThePirateBay.SORT_SEEDS);

                    Map<String, ArrayList<String>> map = new HashMap<>();
                    for(ArrayList<String> torrent : search) {
                        if(!map.containsKey(torrent.get(0))) {
                            map.put(torrent.get(0), torrent);
                        }
                    }
                    torrents = new ArrayList<>(map.values());
                }
            } catch(UnknownHostException e) {
                e.printStackTrace();
                return null;
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }

            float torrentNum = 0;
            float torrentsLength = torrents.size();

            for(ArrayList<String> torrent : torrents) {

                if(isCancelled()) {
                    break;
                }

                TheMovieDb tmdb = new TheMovieDb();
                String torrentTitle = torrent.get(0);
                String torrentYear = torrent.get(1);

                ArrayList<String> tmdbResults = null;
                try {
                    tmdbResults = tmdb.search(torrentTitle, torrentYear);
                } catch (Exception e) {
                    e.printStackTrace();
                    return null;
                }

                if(tmdbResults != null && !tmdbResults.isEmpty()) {
                    String id = tmdbResults.get(0);
                    String title = tmdbResults.get(1);
                    String year = tmdbResults.get(2);
                    String posterSmall = tmdbResults.get(3);
                    String posterMedium = tmdbResults.get(4);
                    String posterLarge = tmdbResults.get(5);
                    String posterXLarge = tmdbResults.get(6);
                    String rating = tmdbResults.get(7);
                    String release = torrent.get(2);
                    String size = torrent.get(5);
                    String seeders = torrent.get(7);
                    String magnetLink = torrent.get(3);

                    results.add(new Movie(Arrays.asList(id, title, year,
                            posterSmall, posterMedium, posterLarge, posterXLarge,
                            rating, release, size, seeders, magnetLink)));
                }

                torrentNum++;
                float progress = torrentNum/torrentsLength * 100;
                publishProgress((int) progress);
            }

            return results;
        }

        protected void onProgressUpdate(Integer... progress) {
            super.onProgressUpdate(progress[0]);
            progressFragment.setProgress(progress[0]);
        }

        protected void onPostExecute(ArrayList<Movie> results) {
            if(progressFragment != null) {
                try {
                    progressFragment.dismiss();
                    progressFragment = null;
                } catch (Exception e) {
                }
            }
            if(results != null && !results.isEmpty()) {
                Collections.sort(results);
                movies = results;
                beginTransaction(results);
            }
        }

    }

}
