package com.bukanir.android.activities;

import android.content.Intent;
import android.os.AsyncTask;
import android.os.Build;
import android.support.v4.app.FragmentManager;
import android.support.v7.app.ActionBar;
import android.support.v7.app.ActionBarActivity;
import android.os.Bundle;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.Window;

import com.bukanir.android.R;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.fragments.MovieFragment;
import com.bukanir.android.utils.Utils;
import com.thinkfree.showlicense.android.ShowLicense;

public class MovieActivity extends ActionBarActivity {

    public static final String TAG = "MovieActivity";

    private Movie movie;
    private MovieTask movieTask;
    private static FragmentManager fragmentManager;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);

        requestWindowFeature(Window.FEATURE_INDETERMINATE_PROGRESS);
        setProgressBarIndeterminateVisibility(false);

        setContentView(R.layout.activity_movie);

        fragmentManager = getSupportFragmentManager();

        final ActionBar actionBar = getSupportActionBar();
        actionBar.setDisplayShowTitleEnabled(true);
        actionBar.setDisplayHomeAsUpEnabled(true);

        if(savedInstanceState != null) {
            movie = (Movie) savedInstanceState.getSerializable("movie");
        } else {
            Bundle bundle = getIntent().getExtras();
            movie = (Movie) bundle.get("movie");

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
                Intent licenses = ShowLicense.createActivityIntent(this, null, Utils.projectList);
                startActivity(licenses);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    private class MovieTask extends AsyncTask<Void, Void, Movie> {

        protected Movie doInBackground(Void... params) {
            while(movie.overview == null) {
                movie.getSummary();
                if(isCancelled()) {
                    break;
                }
            }
            Log.d(TAG, "movie:" + movie.toString());
            return movie;
        }

        protected void onPostExecute(Movie result) {
            fragmentManager.beginTransaction()
                    .add(R.id.container, MovieFragment.newInstance(result))
                    .commit();
        }

    }

}
