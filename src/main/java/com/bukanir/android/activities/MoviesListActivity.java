package com.bukanir.android.activities;

import android.Manifest;
import android.app.AlertDialog;
import android.app.Dialog;
import android.app.DownloadManager;
import android.app.SearchManager;
import android.content.BroadcastReceiver;
import android.content.ComponentName;
import android.content.Context;
import android.content.DialogInterface;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.os.AsyncTask;
import android.os.Build;
import android.preference.PreferenceManager;
import android.support.annotation.NonNull;
import android.support.v4.app.ActivityCompat;
import android.support.v4.app.DialogFragment;
import android.support.v4.app.FragmentTransaction;
import android.support.v4.content.ContextCompat;
import android.support.v4.view.MenuItemCompat;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.SearchView;
import android.os.Bundle;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.AdapterView;
import android.widget.ProgressBar;
import android.widget.Spinner;
import android.widget.Toast;

import com.bukanir.android.application.Settings;
import com.bukanir.android.clients.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MoviesListFragment;
import com.bukanir.android.helpers.Connectivity;
import com.bukanir.android.helpers.Update;
import com.bukanir.android.helpers.Utils;
import com.google.android.gms.analytics.HitBuilders;
import com.google.android.gms.analytics.Tracker;

import java.util.ArrayList;

public class MoviesListActivity extends AppCompatActivity {

    public static final String TAG = "MoviesListActivity";

    private boolean twoPane;

    private ArrayList<Movie> movies;
    private MoviesTask moviesTask;

    private Spinner spinner;
    private ProgressBar progressBar;

    private String cacheDir;

    private boolean userIsInteracting;
    private BroadcastReceiver downloadReceiver;
    private boolean downloadReceiverRegistered;

    private String category = "201";
    private int selectedCategory;

    private Settings settings;

    public static final int RC_PERMISSION_WRITE_EXTERNAL_STORAGE = 313;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        settings = new Settings(this);
        cacheDir = getCacheDir().toString();

        setContentView(R.layout.activity_movie_list);

        spinner = (Spinner) findViewById(R.id.spinner);
        progressBar = (ProgressBar) findViewById(R.id.progressbar);

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        toolbar.setLogo(R.drawable.ic_launcher);
        setSupportActionBar(toolbar);

        getSupportActionBar().setTitle(null);

        if(findViewById(R.id.movie_container) != null) {
            twoPane = true;
        } else {
            twoPane = false;
        }

        Tracker tracker = Utils.getTracker(this);
        tracker.setScreenName("Movies List");
        tracker.send(new HitBuilders.ScreenViewBuilder().build());

        if(Update.checkUpdate(this)) {
            downloadReceiver = Update.getDownloadReceiver(this);
            registerReceiver(downloadReceiver, new IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE));
            downloadReceiverRegistered = true;

            if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                new UpdateTask().executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR);
            } else {
                new UpdateTask().execute();
            }
        }

        int permissionCheck = ContextCompat.checkSelfPermission(MoviesListActivity.this, Manifest.permission.WRITE_EXTERNAL_STORAGE);
        if(permissionCheck != PackageManager.PERMISSION_GRANTED) {
            Log.d(TAG, String.format("permissionCheck:%d", permissionCheck));
            ActivityCompat.requestPermissions(MoviesListActivity.this, new String[]{Manifest.permission.WRITE_EXTERNAL_STORAGE}, RC_PERMISSION_WRITE_EXTERNAL_STORAGE);
        }

        prepareActionBar();

        if(!settings.eulaAccepted()) {
            new EulaFragment().show(getSupportFragmentManager(), "Eula");
        } else {
            if(savedInstanceState != null) {
                movies = savedInstanceState.getParcelableArrayList("movies");
                if(movies != null && !movies.isEmpty()) {
                    try {
                        beginTransaction(movies);
                    } catch(Exception e) {
                        e.printStackTrace();
                    }
                }
            } else {
                startMoviesTask(false);
            }
        }
    }

    @Override
    protected void onPause() {
        Log.d(TAG, "onPause");
        super.onPause();
        if(downloadReceiver != null) {
            if(downloadReceiverRegistered) {
                try {
                    unregisterReceiver(downloadReceiver);
                } catch(Exception e) {
                    e.printStackTrace();
                }
                downloadReceiverRegistered = false;
            }
        }
        cancelMoviesTask();
    }

    @Override
    protected void onResume() {
        Log.d(TAG, "onResume");
        super.onResume();
        if (progressBar != null) {
            progressBar.setVisibility(View.GONE);
        }
        if(settings != null && settings.eulaAccepted()) {
            if(movies == null || movies.isEmpty()) {
                startMoviesTask(false);
            }
        }
    }

    @Override
    public void onSaveInstanceState(Bundle outState) {
        Log.d(TAG, "onSaveInstanceState");
        if(movies != null) {
            outState.putParcelableArrayList("movies", movies);
        }
        if(selectedCategory >= 0) {
            outState.putInt("selectedCategory", selectedCategory);
        }
        super.onSaveInstanceState(outState);
    }

    @Override
    public void onRestoreInstanceState(@NonNull Bundle savedInstanceState) {
        Log.d(TAG, "onRestoreInstanceState");
        if(savedInstanceState.containsKey("selectedCategory")) {
            if(spinner != null) {
                spinner.setSelection(savedInstanceState.getInt("selectedCategory"));
            }
        }
    }

    @Override
    public void onUserInteraction() {
        super.onUserInteraction();
        userIsInteracting = true;
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
            case R.id.action_search:
                onSearchRequested();
                return true;
            case R.id.action_sync:
                if(category != null) {
                    startMoviesTask(true);
                }
                return true;
            case R.id.action_favorites:
                startActivity(new Intent(this, FavoritesActivity.class));
                return true;
            case R.id.action_settings:
                startActivity(new Intent(this, SettingsActivity.class));
                return true;
            case R.id.action_about:
                Utils.showAbout(this);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    @Override
    public void onRequestPermissionsResult(int requestCode,  String permissions[], int[] grantResults) {
        switch(requestCode) {
            case RC_PERMISSION_WRITE_EXTERNAL_STORAGE: {
                if(grantResults.length > 0 && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                    Log.d(TAG, "External storage allowed");
                } else {
                    Log.d(TAG, "External storage denied");
                }
                break;
            }
        }
    }

    public static class EulaFragment extends DialogFragment {

        @Override
        public void onCreate(Bundle savedInstanceState) {
            this.setRetainInstance(true);
            super.onCreate(savedInstanceState);
        }

        @NonNull
        @Override
        public Dialog onCreateDialog(Bundle savedInstanceState) {
            AlertDialog.Builder alertDialogBuilder = new AlertDialog.Builder(getActivity());

            alertDialogBuilder.setMessage(getString(R.string.eula_text));
            alertDialogBuilder.setCancelable(false);

            alertDialogBuilder.setPositiveButton(getString(R.string.eula_positive_button), new DialogInterface.OnClickListener() {
                @Override
                public void onClick(DialogInterface dialog, int which) {
                    SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(getActivity());
                    SharedPreferences.Editor edit = prefs.edit();
                    edit.putBoolean("eula_accepted", true);
                    edit.apply();
                    dialog.dismiss();

                    ((MoviesListActivity) getActivity()).startMoviesTask(false);
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
        getSupportActionBar().setDisplayShowTitleEnabled(true);
        getSupportActionBar().setDisplayHomeAsUpEnabled(false);

        spinner.setOnItemSelectedListener(new AdapterView.OnItemSelectedListener() {

            @Override
            public void onItemSelected(AdapterView<?> adapter, View v, int position, long id) {
                if (position == 0) {
                    category = "201";
                    selectedCategory = 0;
                } else if (position == 1) {
                    category = "207";
                    selectedCategory = 1;
                } else if (position == 2) {
                    category = "205";
                    selectedCategory = 2;
                } else if (position == 3) {
                    category = "208";
                    selectedCategory = 3;
                }

                if(userIsInteracting) {
                    startMoviesTask(false);
                }
            }

            @Override
            public void onNothingSelected(AdapterView<?> arg0) {
            }
        });
    }

    private void startMoviesTask(boolean refresh) {
        String force;
        if(refresh) {
            force = "1";
        } else {
            force = "0";
        }

        moviesTask = new MoviesTask();
        if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
            moviesTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, category, force);
        } else {
            moviesTask.execute(category, force);
        }
    }

    public void cancelMoviesTask() {
        if(moviesTask != null) {
            if(moviesTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                moviesTask.cancel(true);
            }
        }
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

    private class MoviesTask extends AsyncTask<String, Integer, ArrayList<Movie>> {

        String category;

        protected void onPreExecute() {
            super.onPreExecute();
            if(progressBar != null) {
                progressBar.setVisibility(View.VISIBLE);
            }
        }

        protected ArrayList<Movie> doInBackground(String... params) {

            category = params[0];

            String force;
            if(params.length == 2) {
                force = params[1];
            } else {
                force = "0";
            }
            int refresh = Integer.parseInt(force);

            ArrayList<Movie> results;

            try {
                results = BukanirClient.getTopResults(
                        Integer.valueOf(category),
                        settings.listCount(),
                        refresh,
                        cacheDir,
                        settings.cacheDays()
                );
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }

            if(isCancelled()) {
                return null;
            }

            return results;
        }

        protected void onPostExecute(final ArrayList<Movie> results) {
            if(progressBar != null) {
                progressBar.setVisibility(View.GONE);
            }
            if(results != null && !results.isEmpty()) {
                movies = results;
                try {
                    beginTransaction(results);
                } catch(Exception e) {
                    e.printStackTrace();
                }
            } else {
                if(Connectivity.isConnected(getApplicationContext())) {
                    Toast.makeText(getApplicationContext(), getString(R.string.error_text_connection), Toast.LENGTH_SHORT).show();
                } else {
                    Toast.makeText(getApplicationContext(), getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
                }
            }
        }

    }

    private class UpdateTask extends AsyncTask<Void, Void, Boolean> {

        protected void onPreExecute() {
            super.onPreExecute();
        }

        protected Boolean doInBackground(Void... params) {
            return Update.updateExists(getApplication());
        }

        protected void onPostExecute(Boolean result) {
            if(result) {
                Update.showUpdate(MoviesListActivity.this);
            }
        }
    }

}
