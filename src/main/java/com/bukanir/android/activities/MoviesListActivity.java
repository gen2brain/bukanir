package com.bukanir.android.activities;

import android.annotation.SuppressLint;
import android.app.AlertDialog;
import android.app.Dialog;
import android.app.SearchManager;
import android.content.Context;
import android.content.DialogInterface;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.AsyncTask;
import android.os.Build;
import android.preference.PreferenceManager;
import android.support.v4.app.DialogFragment;
import android.support.v4.app.Fragment;
import android.support.v4.app.FragmentTransaction;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.ActionBarActivity;
import android.support.v7.app.ActionBar;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.widget.ArrayAdapter;
import android.widget.Toast;

import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MoviesListFragment;
import com.bukanir.android.fragments.ProgressFragment;
import com.bukanir.android.scrapers.TheMovieDb;
import com.bukanir.android.scrapers.ThePirateBay;
import com.bukanir.android.utils.Cache;
import com.bukanir.android.utils.Utils;
import com.thinkfree.showlicense.android.ShowLicense;

import java.net.UnknownHostException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import io.vov.vitamio.LibsChecker;

public class MoviesListActivity extends ActionBarActivity implements ActionBar.OnNavigationListener {

    public static final String TAG = "MoviesListActivity";

    private boolean twoPane;
    private ArrayList<Movie> movies;
    private MoviesTask moviesTask;
    private String category;
    private int listCount;
    private boolean proxy;

    private static final String STATE_SELECTED_NAVIGATION_ITEM = "selected_navigation_item";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        if(!LibsChecker.checkVitamioLibs(this)) {
            return;
        }

        setContentView(R.layout.activity_movie_list);

        if(findViewById(R.id.movie_container) != null) {
            twoPane = true;
        } else {
            twoPane = false;
        }

        if(savedInstanceState != null) {
            movies = (ArrayList<Movie>) savedInstanceState.getSerializable("movies");
        }

        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
        Boolean eulaAccepted = prefs.getBoolean("eula_accepted", false);

        if(!eulaAccepted) {
            new EulaFragment().show(getSupportFragmentManager(), "Eula");
        } else {
            prepareActionBar();
        }
    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        super.onDestroy();
        cancelMovieTask();
    }

    @Override
    protected void onPause() {
        Log.d(TAG, "onPause");
        super.onPause();
    }

    @Override
    protected void onResume() {
        Log.d(TAG, "onResume");
        super.onResume();
    }

    @Override
    protected void onNewIntent(Intent intent) {
        Log.d(TAG, "onNewIntent");
        super.onNewIntent(intent);
        handleSearchIntent(intent);
    }

    @Override
    public void onRestoreInstanceState(Bundle savedInstanceState) {
        Log.d(TAG, "onRestoreInstanceState");
        if(savedInstanceState.containsKey(STATE_SELECTED_NAVIGATION_ITEM)) {
            getSupportActionBar().setSelectedNavigationItem(
                    savedInstanceState.getInt(STATE_SELECTED_NAVIGATION_ITEM));
        }
    }

    @Override
    public void onSaveInstanceState(Bundle outState) {
        Log.d(TAG, "onSaveInstanceState");
        //super.onSaveInstanceState(outState);
        outState.putSerializable("movies", movies);
        outState.putInt(STATE_SELECTED_NAVIGATION_ITEM,
                getSupportActionBar().getSelectedNavigationIndex());
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.main, menu);
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
            case R.id.action_sync:
                if(category != null) {
                    startMovieTask();
                }
                return true;
            case R.id.action_licenses:
                Intent licenses = ShowLicense.createActivityIntent(this, null, Utils.projectList);
                startActivity(licenses);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    @Override
    public boolean onNavigationItemSelected(int position, long id) {
        Log.d(TAG, "onNavigationItemSelected");
        if(position == 0) {
            category = ThePirateBay.CATEGORY_MOVIES;
        } else if(position == 1) {
            category = ThePirateBay.CATEGORY_HD_MOVIES;
        } else if(position == 2) {
            category = ThePirateBay.CATEGORY_MOVIES_DVDR;
        }

        movies = Cache.getObject(category, getCacheDir(), this);

        if(movies != null) {
            beginTransaction(movies, null);
        } else {
            startMovieTask();
        }

        return true;
    }

    @SuppressLint("ValidFragment")
    public class EulaFragment extends DialogFragment {

        @Override
        public Dialog onCreateDialog(Bundle savedInstanceState) {
            AlertDialog.Builder alertDialogBuilder = new AlertDialog.Builder(getActivity());

            alertDialogBuilder.setMessage(getString(R.string.eula_text));

            alertDialogBuilder.setPositiveButton(getString(R.string.eula_positive_button), new DialogInterface.OnClickListener() {
                @Override
                public void onClick(DialogInterface dialog, int which) {
                    SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(getActivity());
                    SharedPreferences.Editor edit = prefs.edit();
                    edit.putBoolean("eula_accepted", true);
                    edit.commit();
                    dialog.dismiss();

                    prepareActionBar();
                }
            });

            alertDialogBuilder.setNegativeButton(getString(R.string.eula_negative_button), new DialogInterface.OnClickListener() {
                @Override
                public void onClick(DialogInterface dialog, int which) {
                    dialog.dismiss();
                    getActivity().finish();
                }
            });

            return alertDialogBuilder.create();
        }
    }

    private void prepareActionBar() {
        ActionBar actionBar = getSupportActionBar();
        actionBar.setDisplayShowTitleEnabled(true);
        actionBar.setDisplayHomeAsUpEnabled(false);

        actionBar.setNavigationMode(ActionBar.NAVIGATION_MODE_LIST);
        actionBar.setListNavigationCallbacks(
                new ArrayAdapter<String>(
                        actionBar.getThemedContext(),
                        android.R.layout.simple_list_item_1,
                        android.R.id.text1,
                        new String[]{
                                getString(R.string.title_section1),
                                getString(R.string.title_section2),
                                getString(R.string.title_section3),
                        }
                ),
                this
        );
    }

    private void startMovieTask() {
        if(Utils.isNetworkAvailable(this)) {
            SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
            listCount = Integer.valueOf(prefs.getString("list_count", "30"));
            proxy = prefs.getBoolean("proxy", true);

            moviesTask = new MoviesTask();
            if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                moviesTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, null, category);
            } else {
                moviesTask.execute(null, category);
            }
        } else {
            Toast.makeText(this, getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
        }
    }

    public void cancelMovieTask() {
        if(moviesTask != null) {
            if(moviesTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                moviesTask.cancel(true);
            }
        }
    }

    private void beginTransaction(ArrayList<Movie> results, String query) {
        FragmentTransaction ft = getSupportFragmentManager().beginTransaction();
        if(twoPane) {
            if(query != null) {
                Intent intent = new Intent(this, SearchActivity.class);
                intent.putExtra("search", results);
                startActivity(intent);
            } else {
                Fragment prev = getSupportFragmentManager().findFragmentById(R.id.list_container);
                if (prev != null) {
                    ft.remove(prev);
                }
                ft.replace(R.id.list_container, MoviesListFragment.newInstance(results, twoPane));
                ft.commit();
            }
        } else {
            if(query != null) {
                Intent intent = new Intent(this, SearchActivity.class);
                intent.putExtra("search", results);
                startActivity(intent);
            } else {
                Fragment prev = getSupportFragmentManager().findFragmentById(R.id.container);
                if (prev != null) {
                    ft.remove(prev);
                }
                ft.replace(R.id.container, MoviesListFragment.newInstance(results, twoPane));
                ft.commit();
            }
        }
    }

    private void handleSearchIntent(Intent intent) {
        if(Intent.ACTION_SEARCH.equals(intent.getAction())) {
            String query = intent.getStringExtra(SearchManager.QUERY);
            if(Utils.isNetworkAvailable(this)) {
                moviesTask = new MoviesTask();
                if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                    moviesTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, query, null);
                } else {
                    moviesTask.execute(query, null);
                }
            } else {
                Toast.makeText(this, getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
            }
        } else {
            movies = Cache.getObject(category, getCacheDir(), this);
            if(movies != null) {
                beginTransaction(movies, null);
            }
        }
    }

    private class MoviesTask extends AsyncTask<String, Integer, ArrayList<Movie>> {

        String query;
        String category;
        ProgressFragment progressFragment;

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
            category = params[1];
            ArrayList<Movie> results = new ArrayList<>();

            ThePirateBay tpb = new ThePirateBay(proxy);
            List<ArrayList<String>> torrents;

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
                } else {
                    torrents = tpb.top(category);
                    if(torrents.size() >= listCount) {
                        torrents = torrents.subList(0, listCount);
                    }
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
                if(category != null) {
                    movies = results;
                    Cache.saveObject(category, getCacheDir(), results);
                }
                beginTransaction(results, query);
            }
        }

    }

}
