package com.bukanir.android.activities;

import android.app.AlertDialog;
import android.content.Intent;
import android.os.AsyncTask;
import android.os.Build;
import android.support.v4.app.FragmentManager;
import android.support.v7.app.ActionBarActivity;
import android.os.Bundle;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.ProgressBar;

import com.bukanir.android.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.entities.Summary;
import com.bukanir.android.fragments.MovieFragment;
import com.bukanir.android.utils.Utils;
import com.google.android.gms.analytics.HitBuilders;
import com.google.android.gms.analytics.Tracker;
import com.thinkfree.showlicense.android.ShowLicense;

import go.Go;

public class MovieActivity extends ActionBarActivity {

    public static final String TAG = "MovieActivity";

    private Movie movie;
    private MovieTask movieTask;
    private ProgressBar progressBar;
    private static FragmentManager fragmentManager;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        if(!Utils.isX86()) {
            Go.init(getApplicationContext());
        }

        setContentView(R.layout.activity_movie);

        fragmentManager = getSupportFragmentManager();

        progressBar = (ProgressBar) findViewById(R.id.progressbar);

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        toolbar.setLogo(R.drawable.ic_launcher);
        setSupportActionBar(toolbar);

        getSupportActionBar().setDisplayShowTitleEnabled(true);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        if(savedInstanceState != null) {
            movie = (Movie) savedInstanceState.getSerializable("movie");
        } else {
            Bundle bundle = getIntent().getExtras();
            movie = (Movie) bundle.get("movie");

            getSupportActionBar().setSubtitle(movie.title);

            Tracker tracker = Utils.getTracker(this);
            tracker.setScreenName(movie.title);
            tracker.send(new HitBuilders.AppViewBuilder().build());

            movieTask = new MovieTask();
            if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                movieTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR);
            } else {
                movieTask.execute();
            }
        }
    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        super.onDestroy();
        if(movieTask != null) {
            if(movieTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                movieTask.cancel(true);
            }
        }
    }

    @Override
    protected void onSaveInstanceState(Bundle outState) {
        super.onSaveInstanceState(outState);
        outState.putSerializable("movie", movie);
    }


    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.movie, menu);
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
                onBackPressed();
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

    private class MovieTask extends AsyncTask<Void, Void, Summary> {

        protected void onPreExecute() {
            super.onPreExecute();
            if(progressBar != null) {
                progressBar.setVisibility(View.VISIBLE);
            }
        }

        protected Summary doInBackground(Void... params) {
            if(isCancelled()) {
                return null;
            }

            Summary summary = BukanirClient.getSummary(Integer.valueOf(movie.id), Integer.valueOf(movie.category), Integer.valueOf(movie.season));
            return summary;
        }

        protected void onPostExecute(Summary summary) {
            if(progressBar != null) {
                progressBar.setVisibility(View.INVISIBLE);
            }
            fragmentManager.beginTransaction()
                .add(R.id.container, MovieFragment.newInstance(movie, summary))
                .commit();
        }

    }

}
