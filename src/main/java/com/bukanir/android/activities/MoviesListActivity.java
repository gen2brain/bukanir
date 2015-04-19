package com.bukanir.android.activities;

import android.annotation.SuppressLint;
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
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.AdapterView;
import android.widget.ProgressBar;
import android.widget.Spinner;
import android.widget.Toast;

import com.bukanir.android.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MoviesListFragment;
import com.bukanir.android.services.BukanirHttpService;
import com.bukanir.android.utils.Connectivity;
import com.bukanir.android.utils.Update;
import com.bukanir.android.utils.Utils;
import com.google.android.gms.analytics.HitBuilders;
import com.google.android.gms.analytics.Tracker;
import com.thinkfree.showlicense.android.ShowLicense;

import java.util.ArrayList;

import go.Go;
import io.vov.vitamio.LibsChecker;

public class MoviesListActivity extends ActionBarActivity {

    public static final String TAG = "MoviesListActivity";

    private boolean twoPane;
    private ArrayList<Movie> movies;
    private MoviesTask moviesTask;
    private int listCount;
    private Spinner spinner;
    private ProgressBar progressBar;
    private String cacheDir;

    private BroadcastReceiver downloadReceiver;

    private String category = "201";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        if(!LibsChecker.checkVitamioLibs(this)) {
            return;
        }

        Go.init(getApplicationContext());

        cacheDir = getCacheDir().toString();

        if(Utils.isX86()) {
            if(!Utils.isHttpServiceRunning(this)) {
                Intent intent = new Intent(this, BukanirHttpService.class);
                startService(intent);
            }
        }

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
        tracker.send(new HitBuilders.AppViewBuilder().build());

        if(Update.checkUpdate(this)) {
            downloadReceiver = Update.getDownloadReceiver(this);
            registerReceiver(downloadReceiver, new IntentFilter(DownloadManager.ACTION_DOWNLOAD_COMPLETE));

            if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                new UpdateTask().executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR);
            } else {
                new UpdateTask().execute();
            }
        }

        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
        Boolean eulaAccepted = prefs.getBoolean("eula_accepted", false);

        if(!eulaAccepted) {
            new EulaFragment().show(getSupportFragmentManager(), "Eula");
        } else {
            if(savedInstanceState != null) {
                movies = (ArrayList<Movie>) savedInstanceState.getSerializable("movies");
                if(movies != null && !movies.isEmpty()) {
                    try {
                        beginTransaction(movies);
                    } catch(Exception e) {
                        e.printStackTrace();
                    }
                } else {
                    prepareActionBar();
                }
            } else {
                prepareActionBar();
            }
        }
    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        if(downloadReceiver != null) {
            unregisterReceiver(downloadReceiver);
        }
        super.onDestroy();
        cancelMovieTask();

        if(Utils.isX86()) {
            if(Utils.isHttpServiceRunning(this)) {
                Intent intent = new Intent(this, BukanirHttpService.class);
                stopService(intent);
            }
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
    public void onRestoreInstanceState(Bundle savedInstanceState) {
        Log.d(TAG, "onRestoreInstanceState");
    }

    @Override
    public void onSaveInstanceState(Bundle outState) {
        Log.d(TAG, "onSaveInstanceState");
        //super.onSaveInstanceState(outState);
        outState.putSerializable("movies", movies);
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

        spinner.setOnItemSelectedListener(new AdapterView.OnItemSelectedListener() {

            @Override
            public void onItemSelected(AdapterView<?> adapter, View v, int position, long id) {
                if(position == 0) {
                    category = "201";
                } else if(position == 1) {
                    category = "207";
                } else if(position == 2) {
                    category = "205";
                }

                startMovieTask(false);
            }

            @Override
            public void onNothingSelected(AdapterView<?> arg0) {
            }
        });

        startMovieTask(false);
    }

    private void startMovieTask(boolean refresh) {
        if(Connectivity.isConnected(this)) {
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
                moviesTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, category, force);
            } else {
                moviesTask.execute(category, force);
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

    private void beginTransaction(ArrayList<Movie> results) {
        FragmentTransaction ft = getSupportFragmentManager().beginTransaction();
        if(twoPane) {
            Fragment prev = getSupportFragmentManager().findFragmentById(R.id.list_container);
            if (prev != null) {
                ft.remove(prev);
            }
            ft.replace(R.id.list_container, MoviesListFragment.newInstance(results, twoPane));
            ft.commit();
        } else {
            Fragment prev = getSupportFragmentManager().findFragmentById(R.id.container);
            if (prev != null) {
                ft.remove(prev);
            }
            ft.replace(R.id.container, MoviesListFragment.newInstance(results, twoPane));
            ft.commit();
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
                results = BukanirClient.getTopMovies(category, listCount, refresh, cacheDir);
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }

            return results;
        }

        protected void onPostExecute(final ArrayList<Movie> results) {
            if(progressBar != null) {
                progressBar.setVisibility(View.INVISIBLE);
            }
            if(results != null && !results.isEmpty()) {
                movies = results;
                try {
                    beginTransaction(results);
                } catch(Exception e) {
                    e.printStackTrace();
                }
            } else {
                Toast.makeText(getApplicationContext(), getString(R.string.error_text_connection), Toast.LENGTH_SHORT).show();
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
                Utils.showUpdate(MoviesListActivity.this);
            }
        }
    }

}
