package com.bukanir.android.activities;

import android.content.Intent;
import android.os.AsyncTask;
import android.os.Bundle;
import android.support.v4.app.FragmentTransaction;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.ProgressBar;
import android.widget.Toast;

import com.bukanir.android.clients.BukanirClient;
import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.entities.Summary;
import com.bukanir.android.fragments.MovieFragment;
import com.bukanir.android.helpers.Connectivity;
import com.bukanir.android.helpers.Dialogs;
import com.bukanir.android.helpers.Utils;
import com.google.android.gms.analytics.HitBuilders;
import com.google.android.gms.analytics.Tracker;

public class MovieActivity extends AppCompatActivity {

    public static final String TAG = "MovieActivity";

    private Movie movie;
    private MovieTask movieTask;
    private ProgressBar progressBar;

    Summary summary;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        setContentView(R.layout.activity_movie);

        progressBar = (ProgressBar) findViewById(R.id.progressbar);

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        toolbar.setLogo(R.drawable.ic_launcher);
        setSupportActionBar(toolbar);

        if(getSupportActionBar() != null) {
            getSupportActionBar().setDisplayHomeAsUpEnabled(true);
        }

        if(savedInstanceState != null) {
            movie = (Movie) savedInstanceState.getSerializable("movie");
            getSupportActionBar().setTitle(movie.title);
        } else {
            Bundle bundle = getIntent().getExtras();
            movie = (Movie) bundle.getSerializable("movie");
            getSupportActionBar().setTitle(movie.title);

            startMovieTask();
        }
    }

    @Override
    protected void onPause() {
        Log.d(TAG, "onPause");
        super.onPause();
        cancelMovieTask();
    }

    @Override
    protected void onSaveInstanceState(Bundle outState) {
        if(movie != null) {
            outState.putSerializable("movie", movie);
        }
        super.onSaveInstanceState(outState);
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
                startActivity(new Intent(this, SettingsActivity.class));
                return true;
            case R.id.action_search:
                onSearchRequested();
                return true;
            case R.id.action_sync:
                if(movie != null) {
                    startMovieTask();
                }
                return true;
            case android.R.id.home:
                onBackPressed();
                return true;
            case R.id.action_about:
                Dialogs.showAbout(this);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    private void startMovieTask() {
        if(Connectivity.isConnected(this)) {
            Tracker tracker = Utils.getTracker(this);
            tracker.setScreenName(movie.title);
            tracker.send(new HitBuilders.ScreenViewBuilder().build());

            movieTask = new MovieTask();
            movieTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR);
        } else {
            Toast.makeText(this, getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
        }
    }

    public void cancelMovieTask() {
        if(movieTask != null) {
            if(movieTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                movieTask.cancel(true);
                BukanirClient.cancel();
            }
        }
    }

    private void beginTransaction() {
        FragmentTransaction ft;
        ft = getSupportFragmentManager().beginTransaction();
        ft.replace(R.id.container, MovieFragment.newInstance(movie, summary));
        ft.commitAllowingStateLoss();
    }

    private class MovieTask extends AsyncTask<Void, Void, Summary> {

        protected void onPreExecute() {
            super.onPreExecute();
            if(progressBar != null) {
                progressBar.setVisibility(View.VISIBLE);
            }
        }

        protected Summary doInBackground(Void... params) {
            Summary s = BukanirClient.getSummary(
                    Integer.valueOf(movie.id),
                    Integer.valueOf(movie.category),
                    Integer.valueOf(movie.season),
                    Integer.valueOf(movie.episode));

            if(isCancelled()) {
                return null;
            }
            return s;
        }

        protected void onPostExecute(Summary s) {
            if(progressBar != null) {
                progressBar.setVisibility(View.GONE);
            }

            if(s != null && movie != null) {
                summary = s;
                try {
                    beginTransaction();
                } catch(Exception e) {
                    e.printStackTrace();
                }
            }
        }

    }

}
