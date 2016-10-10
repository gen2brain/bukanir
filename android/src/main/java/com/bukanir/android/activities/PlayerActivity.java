package com.bukanir.android.activities;

import android.content.Context;
import android.net.Uri;
import android.net.wifi.WifiManager;
import android.os.AsyncTask;
import android.os.Bundle;
import android.os.Handler;
import android.support.v7.app.ActionBar;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.text.Html;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.SurfaceHolder;
import android.view.View;
import android.widget.ProgressBar;
import android.widget.SeekBar;
import android.widget.TextView;
import android.widget.Toast;

import com.bukanir.android.R;
import com.bukanir.android.application.Settings;
import com.bukanir.android.clients.BukanirClient;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.entities.Subtitle;
import com.bukanir.android.entities.TorrentFile;
import com.bukanir.android.helpers.Storage;
import com.bukanir.android.subtitles.Caption;
import com.bukanir.android.subtitles.FormatASS;
import com.bukanir.android.subtitles.FormatSRT;
import com.bukanir.android.subtitles.TimedTextObject;
import com.bukanir.android.widget.media.AndroidMediaController;
import com.bukanir.android.widget.media.IjkVideoView;
import com.bukanir.android.widget.media.MeasureHelper;

import java.io.File;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Locale;

import tv.danmaku.ijk.media.player.IMediaPlayer;
import tv.danmaku.ijk.media.player.IjkMediaPlayer;

public class PlayerActivity extends AppCompatActivity implements SurfaceHolder.Callback {

    private static final String TAG = "PlayerActivity";

    private IjkVideoView videoView;
    private TextView toastTextView;
    private AndroidMediaController mediaController;
    private ProgressBar progressBar;

    private Settings settings;
    private WifiManager.WifiLock wifiLock;

    Movie movie;
    String trailerID;
    String trailerURL;
    TorrentFile torrentFile;

    private int retry = 1;
    private int savedPosition = 0;

    String subtitleDir;
    ArrayList<Subtitle> subtitles;
    int subtitleDelay = 0;
    int subtitleCurrent = 0;

    Collection<Caption> captions;
    public TimedTextObject timedTextObject;
    private TextView subtitleTextView;
    private SubtitleTask subtitleTask;

    private Handler subtitleHandler = new Handler();

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        setContentView(R.layout.activity_player);

        settings = new Settings(this);

        Bundle bundle = getIntent().getExtras();
        movie = (Movie) bundle.get("movie");
        subtitles = (ArrayList<Subtitle>) bundle.get("subtitles");
        torrentFile = (TorrentFile) bundle.get("torrent-file");
        trailerID = (String) bundle.get("trailer-id");
        trailerURL = (String) bundle.get("trailer-url");

        Toolbar toolbar = (Toolbar) findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);

        setTitle();

        ActionBar actionBar = getSupportActionBar();

        mediaController = new AndroidMediaController(this, false);
        mediaController.setSupportActionBar(actionBar);

        subtitleTextView = (TextView) findViewById(R.id.subtitle_view);
        toastTextView = (TextView) findViewById(R.id.toast_text_view);

        progressBar = (ProgressBar) findViewById(R.id.progress_bar);
        progressBar.setVisibility(View.VISIBLE);

        subtitleDir = Storage.getStorage(this) + File.separator + "bukanir" + File.separator + "subtitles";

        initVideoView();
    }

    @Override
    protected void onPause() {
        Log.d(TAG, "onPause");
        super.onPause();
        if(subtitleHandler != null) {
            subtitleHandler.removeCallbacks(subtitleProcessesor);

            if(subtitleTask != null) {
                subtitleTask.cancel(true);
            }
        }

        if(videoView != null) {
            savedPosition = videoView.getCurrentPosition();
            if(videoView.isPlaying()) {
                videoView.stopPlayback();
            }
            videoView.release(true);
        }

        IjkMediaPlayer.native_profileEnd();
    }

    @Override
    protected void onResume() {
        Log.d(TAG, "onPause");
        super.onResume();
        if(videoView != null && savedPosition > 0) {
            if(settings.seek()) {
                videoView.seekTo(savedPosition);
            }
            savedPosition = 0;
        }
    }

    @Override
    public void surfaceCreated(SurfaceHolder surfaceHolder) {
        int wifiLockMode;
        if(settings.wifiHigh()) {
            wifiLockMode = WifiManager.WIFI_MODE_FULL_HIGH_PERF;
        } else {
            wifiLockMode = WifiManager.WIFI_MODE_FULL;
        }
        wifiLock = ((WifiManager) this.getSystemService(Context.WIFI_SERVICE))
                .createWifiLock(wifiLockMode, "lock");
    }

    @Override
    public void surfaceChanged(SurfaceHolder surfaceHolder, int i, int i1, int i2) {
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder surfaceHolder) {
        if(wifiLock != null) {
            if(wifiLock.isHeld()) {
                wifiLock.release();
                wifiLock = null;
            }
        }
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.player, menu);

        if(subtitles == null || subtitles.isEmpty()) {
            menu.findItem(R.id.action_toggle_subtitles).setVisible(false);
            menu.findItem(R.id.action_sub_delay_plus).setVisible(false);
            menu.findItem(R.id.action_sub_delay_minus).setVisible(false);
        } else {
            MenuItem item = menu.findItem(R.id.action_toggle_subtitles);
            if(subtitleCurrent == -1) {
                item.setIcon(R.drawable.ic_subtitles_off);
            } else {
                item.setIcon(R.drawable.ic_subtitles);
            }
        }
        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();
        if(id == R.id.action_toggle_ratio) {
            int aspectRatio = videoView.toggleAspectRatio();
            String aspectRatioText = MeasureHelper.getAspectRatioText(this, aspectRatio);
            toastTextView.setText(aspectRatioText);
            mediaController.showOnce(toastTextView);
            return true;
        } else if(id == R.id.action_toggle_subtitles) {
            if(subtitles != null && !subtitles.isEmpty()) {
                toggleSubtitles();
            }
        } else if(id == R.id.action_sub_delay_plus) {
            subtitleDelay += 1;
            toastTextView.setText(String.format(Locale.ROOT, "Sub delay: %ds", subtitleDelay));
            mediaController.showOnce(toastTextView);
        } else if(id == R.id.action_sub_delay_minus) {
            subtitleDelay -= 1;
            toastTextView.setText(String.format(Locale.ROOT, "Sub delay: %ds", subtitleDelay));
            mediaController.showOnce(toastTextView);
        }

        return super.onOptionsItemSelected(item);
    }

    private void setTitle() {
        String dataTitle = "";
        if(torrentFile != null) {
            if(movie.category.equals("205") || movie.category.equals("208")) {
                int season = Integer.valueOf(movie.season);
                int episode = Integer.valueOf(movie.episode);
                dataTitle = String.format(Locale.ROOT, "%s (S%02dE%02d)", movie.title, season, episode);
            } else {
                dataTitle = String.format("%s (%s)", movie.title, movie.year);
            }
        } else if(trailerURL != null && !trailerURL.isEmpty() && !trailerURL.equals("null")) {
            dataTitle = String.format("%s (%s) - Trailer", movie.title, movie.year);
        }

        if(getSupportActionBar() != null) {
            getSupportActionBar().setTitle(dataTitle);
        }
    }

    private void setSeekVisibility() {
        if(!settings.seek() && trailerURL == null) {
            int progressId = getResources().getIdentifier("mediacontroller_progress", "id", "android");
            SeekBar seekBar = (SeekBar) mediaController.findViewById(progressId);
            if(seekBar != null) {
                seekBar.setVisibility(View.INVISIBLE);
            }
        }
    }

    private void initVideoView() {
        IjkMediaPlayer.loadLibrariesOnce(null);
        IjkMediaPlayer.native_profileBegin("libijkplayer.so");

        videoView = (IjkVideoView) findViewById(R.id.video_view);
        videoView.setMediaController(mediaController);

        subtitleTextView.setTextSize(Integer.valueOf(settings.subtitleSize()));

        videoView.getSurfaceRenderView().getHolder().addCallback(this);

        IMediaPlayer.OnPreparedListener preparedListener = new IMediaPlayer.OnPreparedListener() {
            public void onPrepared(IMediaPlayer mp) {
                if(progressBar != null) {
                    progressBar.setVisibility(View.GONE);
                }

                setSeekVisibility();

                if(subtitles != null && !subtitles.isEmpty()) {
                    subtitleTask = new SubtitleTask();
                    subtitleTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, 0);
                }

                wifiLock.acquire();
            }
        };

        IMediaPlayer.OnCompletionListener completionListener = new IMediaPlayer.OnCompletionListener() {
            @Override
            public void onCompletion(IMediaPlayer mp) {
                onBackPressed();
            }
        };

        IMediaPlayer.OnInfoListener infoListener = new IMediaPlayer.OnInfoListener() {
            @Override
            public boolean onInfo(IMediaPlayer mp, int arg1, int arg2) {
                if(arg1 == IMediaPlayer.MEDIA_INFO_BUFFERING_START) {
                    if(progressBar != null) {
                        progressBar.setVisibility(View.VISIBLE);
                    }
                } else if(arg1 == IMediaPlayer.MEDIA_INFO_BUFFERING_END) {
                    if(progressBar != null) {
                        progressBar.setVisibility(View.GONE);
                    }
                }
                return true;
            }
        };

        IMediaPlayer.OnErrorListener errorListener = new IMediaPlayer.OnErrorListener() {
            @Override
            public boolean onError(IMediaPlayer mp, int framework_err, int impl_err) {
                Log.d(TAG, "onError");

                if(trailerURL != null) {
                    if(retry >= 3) {
                        onBackPressed();
                        Toast.makeText(getApplicationContext(), R.string.error_text_connection, Toast.LENGTH_LONG).show();
                    } else {
                        Log.d(TAG, "retry " + String.valueOf(retry));
                        new TrailerTask().execute(trailerID);
                    }
                } else {
                    onBackPressed();
                    Toast.makeText(getApplicationContext(), R.string.error_text_connection, Toast.LENGTH_LONG).show();
                }
                return true;
            }
        };

        videoView.setOnPreparedListener(preparedListener);
        videoView.setOnCompletionListener(completionListener);
        videoView.setOnInfoListener(infoListener);
        videoView.setOnErrorListener(errorListener);

        String dataURI = "";
        if(torrentFile != null) {
            dataURI = torrentFile.url;
        } else if(trailerURL != null && !trailerURL.isEmpty() && !trailerURL.equals("null")) {
            dataURI = trailerURL;
        }

        if(dataURI != null) {
            videoView.setVideoURI(Uri.parse(dataURI));
        } else {
            Log.e(TAG, "Null dataURI");
            finish();
            return;
        }
        videoView.start();
    }

    private void toggleSubtitles() {
        int count;
        if(subtitles.size() > 3) {
            count = 2;
        } else {
            count = subtitles.size()-1;
        }

        subtitleCurrent++;
        if(subtitleCurrent > count) {
            subtitleCurrent = -1;
        }

        if(subtitleHandler != null) {
            subtitleHandler.removeCallbacks(subtitleProcessesor);
            if(subtitleTask != null) {
                subtitleTask.cancel(true);
            }

            invalidateOptionsMenu();
            if(subtitleCurrent != -1) {
                subtitleTask = new SubtitleTask();
                subtitleTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, subtitleCurrent);
            } else {
                subtitleTextView.setVisibility(View.INVISIBLE);
                toastTextView.setText(getString(R.string.sub_disabled));
                mediaController.showOnce(toastTextView);
            }
        }
    }

    private Runnable subtitleProcessesor = new Runnable() {
        @Override
        public void run() {
            if(videoView != null && videoView.isPlaying()) {
                int currentPos = videoView.getCurrentPosition();
                for(Caption caption : captions) {
                    if(currentPos >= caption.start.getMseconds()+subtitleDelay*1000 &&
                            currentPos <= caption.end.getMseconds()+subtitleDelay*1000) {
                        onTimedText(caption);
                        break;
                    } else if(currentPos > caption.end.getMseconds()+subtitleDelay*1000) {
                        onTimedText(null);
                    }
                }
            }
            subtitleHandler.postDelayed(this, 500);
        }
    };

    public void onTimedText(Caption text) {
        if(text == null) {
            subtitleTextView.setVisibility(View.INVISIBLE);
            return;
        }
        subtitleTextView.setText(Html.fromHtml(text.content));
        subtitleTextView.setVisibility(View.VISIBLE);
    }

    public class SubtitleTask extends AsyncTask<Integer, Void, Subtitle> {

        @Override
        protected void onPreExecute() {
            super.onPreExecute();
        }

        @Override
        protected Subtitle doInBackground(Integer... params) {
            try {
                Subtitle subtitle = subtitles.get(params[0]);
                String subtitleFile = BukanirClient.unzipSubtitle(subtitle.downloadLink, subtitleDir);

                if(subtitleFile.toLowerCase().endsWith(".srt")) {
                    timedTextObject = new FormatSRT().parseFile(subtitleFile);
                } else if(subtitleFile.toLowerCase().endsWith(".ass") || subtitleFile.toLowerCase().endsWith(".ssa")) {
                    timedTextObject = new FormatASS().parseFile(subtitleFile);
                }

                if(timedTextObject != null) {
                    captions = timedTextObject.captions.values();
                }

                return subtitle;
            } catch(Exception e) {
                e.printStackTrace();
                return null;
            }
        }

        @Override
        protected void onPostExecute(Subtitle s) {
            if(timedTextObject != null) {
                subtitleTextView.setText("");
                if(subtitleHandler != null) {
                    subtitleHandler.post(subtitleProcessesor);
                }
            }
            toastTextView.setText(String.format(Locale.ROOT, "Subtitle %d", subtitleCurrent+1));
            mediaController.showOnce(toastTextView);
            super.onPostExecute(s);
        }
    }

    private class TrailerTask extends AsyncTask<String, Void, String> {

        protected String doInBackground(String... params) {
            String video = params[0];
            String result = BukanirClient.getTrailer(video);

            if(isCancelled()) {
                return null;
            }

            return result;
        }

        protected void onPostExecute(String url) {
            retry += 1;
            if(url != null && !url.isEmpty() && !url.equals("empty")) {
                if(videoView != null) {
                    videoView.setVideoURI(Uri.parse(url));
                    videoView.start();
                }
            }
        }

    }

}
