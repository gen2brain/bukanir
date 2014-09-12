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
import android.os.Handler;
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
import android.view.Window;
import android.widget.ArrayAdapter;
import android.widget.Toast;

import com.bukanir.android.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MoviesListFragment;
import com.bukanir.android.services.BukanirHttpService;
import com.bukanir.android.utils.Utils;
import com.thinkfree.showlicense.android.ShowLicense;

import java.util.ArrayList;

import io.vov.vitamio.LibsChecker;

public class MoviesListActivity extends ActionBarActivity implements ActionBar.OnNavigationListener {

    public static final String TAG = "MoviesListActivity";

    private boolean twoPane;
    private ArrayList<Movie> movies;
    private MoviesTask moviesTask;
    private String category;
    private int listCount;

    private static final String STATE_SELECTED_NAVIGATION_ITEM = "selected_navigation_item";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        if(!LibsChecker.checkVitamioLibs(this)) {
            return;
        }

        if(!Utils.isHttpServiceRunning(this)) {
            Intent intent = new Intent(this, BukanirHttpService.class);
            startService(intent);
        }

        supportRequestWindowFeature(Window.FEATURE_INDETERMINATE_PROGRESS);

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

        if(Utils.isHttpServiceRunning(this)) {
            Intent intent = new Intent(this, BukanirHttpService.class);
            stopService(intent);
        }
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
                    startMovieTask(true);
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
            category = "201";
        } else if(position == 1) {
            category = "207";
        } else if(position == 2) {
            category = "202";
        }

        startMovieTask(false);

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
                new ArrayAdapter<>(
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

    private void startMovieTask(boolean refresh) {
        if(Utils.isNetworkAvailable(this)) {
            SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
            listCount = Integer.valueOf(prefs.getString("list_count", "30"));

            String force;
            if(refresh) {
                force = "1";
            } else {
                force = "0";
            }

            moviesTask = new MoviesTask();
            if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                moviesTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, null, category, force);
            } else {
                moviesTask.execute(null, category, force);
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
        Log.d(TAG, "handleSearchIntent");
        if(Intent.ACTION_SEARCH.equals(intent.getAction())) {
            String query = intent.getStringExtra(SearchManager.QUERY);
            if(Utils.isNetworkAvailable(this)) {
                Log.d(TAG, "networkAvailable");
                moviesTask = new MoviesTask();
                if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                    moviesTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, query, null);
                } else {
                    moviesTask.execute(query, null);
                }
            } else {
                Toast.makeText(this, getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
            }
        }
    }

    private class MoviesTask extends AsyncTask<String, Integer, ArrayList<Movie>> {

        String query;
        String category;

        protected void onPreExecute() {
            super.onPreExecute();
            setSupportProgressBarIndeterminateVisibility(true);
        }

        protected ArrayList<Movie> doInBackground(String... params) {

            query = params[0];
            category = params[1];

            String force;
            if(params.length == 3) {
                force = params[2];
            } else {
                force = "0";
            }
            ArrayList<Movie> results;

            boolean refresh = false;
            if(force.equals("1")) {
                refresh = true;
            }

            try {
                Thread.sleep(500);
            } catch (InterruptedException e) {
                e.printStackTrace();
            }

            try {
                if(query != null) {
                    results = BukanirClient.getSearchMovies(query, -1);
                } else {
                    results = BukanirClient.getTopMovies(category, listCount, refresh);
                }
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }

            return results;
        }

        protected void onPostExecute(final ArrayList<Movie> results) {
            setSupportProgressBarIndeterminateVisibility(false);
            if(results != null && !results.isEmpty()) {
                if(category != null) {
                    movies = results;
                }
                try {
                    beginTransaction(results, query);
                } catch(Exception e) {
                    e.printStackTrace();
                }
            }
        }

    }

}
