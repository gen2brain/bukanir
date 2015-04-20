package com.bukanir.android.activities;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.res.Configuration;
import android.graphics.PixelFormat;
import android.net.wifi.WifiManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.PowerManager;
import android.preference.PreferenceManager;
import android.util.DisplayMetrics;
import android.util.Log;
import android.view.MotionEvent;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageButton;
import android.widget.ProgressBar;
import android.widget.TextView;
import android.widget.Toast;

import com.bukanir.android.R;
import com.bukanir.android.Torrent2Http;
import com.bukanir.android.entities.Subtitle;
import com.bukanir.android.services.Torrent2HttpService;
import com.bukanir.android.utils.Utils;

import java.io.IOException;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;

import io.vov.vitamio.MediaPlayer;
import io.vov.vitamio.MediaPlayer.OnPreparedListener;
import io.vov.vitamio.MediaPlayer.OnCompletionListener;
import io.vov.vitamio.MediaPlayer.OnTimedTextListener;
import io.vov.vitamio.MediaPlayer.OnErrorListener;
import io.vov.vitamio.MediaPlayer.OnVideoSizeChangedListener;
import io.vov.vitamio.widget.MediaController;

public class PlayerActivity extends Activity implements SurfaceHolder.Callback,
       MediaController.MediaPlayerControl, OnPreparedListener, OnCompletionListener,
       OnTimedTextListener, OnErrorListener, OnVideoSizeChangedListener {

           private static final String TAG = "PlayerActivity";

           private MediaPlayer mediaPlayer;
           private MediaController mediaController;
           private TextView subtitleView;
           private SurfaceView surfaceView;
           private ImageButton subtitleToggle;
           private WifiManager.WifiLock wifiLock;
           private ProgressBar progressBar;

           String file;
           String sub;
           Subtitle subtitle;
           String subtitleEncoding;

           private long savedPosition = 0;
           private boolean subtitlesEnabled = false;
           private Handler handler = new Handler();

           @Override
           public void onCreate(Bundle savedInstanceState) {
               Log.d(TAG, "onCreate");
               super.onCreate(savedInstanceState);

               Bundle bundle = getIntent().getExtras();
               file = (String) bundle.get("file");
               sub = (String) bundle.get("sub");
               subtitle = (Subtitle) bundle.getSerializable("subtitle");

               if(sub != null && file != null && subtitle != null) {
                   Log.d(TAG, "sub:" + sub.toString());
                   Log.d(TAG, "file:" + file.toString());
                   Log.d(TAG, "subtitle:" + subtitle.toString());
               }

               setContentView(R.layout.player);
               surfaceView = (SurfaceView) findViewById(R.id.surface);
               subtitleView = (TextView) findViewById(R.id.sub1);

               surfaceView.getHolder().setFormat(PixelFormat.RGB_565);
               surfaceView.getHolder().addCallback(this);

               progressBar = (ProgressBar) findViewById(R.id.progress_bar);
               progressBar.setVisibility(View.VISIBLE);

               SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
               subtitleEncoding = prefs.getString("sub_enc", "UTF-8");
           }

           @Override
           protected void onDestroy() {
               Log.d(TAG, "onDestroy");
               super.onDestroy();
               playerRelease();

               if(Utils.isTorrentServiceRunning(this)) {
                   Intent intent = new Intent(this, Torrent2HttpService.class);
                   stopService(intent);
               }
           }

           @Override
           public void surfaceCreated(SurfaceHolder holder) {
               Log.d(TAG, "surfaceCreated");

               if(mediaPlayer == null) {
                   playerPrepare();
               } else {
                   mediaPlayer.setDisplay(surfaceView.getHolder());
               }

               SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
               Boolean wifiHigh = prefs.getBoolean("wifi_high", true);

               int wifiLockMode;
               if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB_MR1) {
                   if(wifiHigh) {
                       wifiLockMode = WifiManager.WIFI_MODE_FULL_HIGH_PERF;
                   } else {
                       wifiLockMode = WifiManager.WIFI_MODE_FULL;
                   }
               } else {
                   wifiLockMode = WifiManager.WIFI_MODE_FULL;
               }
               wifiLock = ((WifiManager) getSystemService(Context.WIFI_SERVICE))
                   .createWifiLock(wifiLockMode, "lock");
           }

           @Override
           public void surfaceDestroyed(SurfaceHolder holder) {
               Log.d(TAG, "surfaceDestroyed");
               pause();
               if(wifiLock != null) {
                   if(wifiLock.isHeld()) {
                       wifiLock.release();
                       wifiLock = null;
                   }
               }
           }

           @Override
           public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
               Log.d(TAG, "surfaceChanged");
               if(mediaController != null) {
                   if(mediaController.isShowing()) {
                       mediaController.hide();
                   }
               }
           }

           @Override
           public void onPrepared(MediaPlayer player) {
               Log.d(TAG, "onPrepared");

               if(progressBar != null) {
                   progressBar.setVisibility(View.INVISIBLE);
               }

               playerStart();
               subtitleTogglePrepare();

               if(sub == null) {
                   subtitleToggle.setVisibility(View.INVISIBLE);
               }

               wifiLock.acquire();
               mediaPlayer.setScreenOnWhilePlaying(true);
               mediaPlayer.setWakeMode(getApplicationContext(), PowerManager.PARTIAL_WAKE_LOCK);
           }

           @Override
           public void onCompletion(MediaPlayer player) {
               Log.d(TAG, "onCompletion");
               playerRelease();
               onBackPressed();
           }

           @Override
           public void onConfigurationChanged(Configuration newConfig) {
               Log.d(TAG, "onConfigurationChanged");
               super.onConfigurationChanged(newConfig);
               if(mediaController.isShowing()) {
                   mediaController.hide();
               }
               setVideoSize();
           }

           @Override
           public void onTimedText(String text) {
               Log.d(TAG, "onTimedText");
               subtitleView.setText(text);
           }

           @Override
           public void onTimedTextUpdate(byte[] pixels, int width, int height) {
               Log.d(TAG, "onTimedTextUpdate:" + pixels.toString());
           }

           @Override
           public boolean onTouchEvent(MotionEvent event) {
               if(event.getAction() == MotionEvent.ACTION_UP) {
                   Log.d(TAG, "onTouchEvent");
                   toggleMediaControllerVisibility();
               }
               return false;
           }

           @Override
           protected void onStop() {
               Log.d(TAG, "onStop");
               super.onStop();
           }

           @Override
           protected void onPause() {
               Log.d(TAG, "onPause");
               super.onPause();
               if(mediaPlayer != null) {
                   savedPosition = mediaPlayer.getCurrentPosition();
                   if(mediaPlayer.isPlaying()) {
                       mediaPlayer.stop();
                   }
               }
           }

           @Override
           protected void onResume() {
               Log.d(TAG, "onResume");
               super.onResume();
               if(mediaPlayer != null) {
                   if(savedPosition > 0) {
                       mediaPlayer.seekTo(savedPosition);
                       savedPosition = 0;
                       mediaPlayer.start();
                   }
               }
           }

           @Override
           public void onVideoSizeChanged(MediaPlayer player, int width, int height) {
               Log.d(TAG, "onVideoSizeChanged");
               setVideoSize();
           }

           @Override
           public void start() {
               mediaPlayer.start();
           }

           @Override
           public void pause() {
               if(mediaPlayer != null) {
                   mediaPlayer.pause();
               }
           }

           @Override
           public long getDuration() {
               return mediaPlayer.getDuration();
           }

           @Override
           public long getCurrentPosition() {
               return mediaPlayer.getCurrentPosition();
           }

           @Override
           public void seekTo(long i) {
               mediaPlayer.seekTo(i);
           }

           @Override
           public boolean isPlaying() {
               return mediaPlayer.isPlaying();
           }

           @Override
           public int getBufferPercentage() {
               return (int) ((mediaPlayer.getCurrentPosition() * 100) / mediaPlayer.getDuration());
           }

           @Override
           public boolean onError(MediaPlayer mediaPlayer, int what, int extra) {
               Log.e(TAG, "onError");
               String error;
               switch(extra) {
                   case MediaPlayer.MEDIA_ERROR_IO:
                   case MediaPlayer.MEDIA_ERROR_MALFORMED:
                   case MediaPlayer.MEDIA_ERROR_TIMED_OUT:
                       error = getString(R.string.error_text_connection);
                       break;

                   case MediaPlayer.MEDIA_ERROR_UNSUPPORTED:
                   case MediaPlayer.MEDIA_ERROR_NOT_VALID_FOR_PROGRESSIVE_PLAYBACK:
                       error = getString(R.string.error_text_unsupported);
                       break;

                   default:
                       error = getString(R.string.error_text_unknown);
                       break;
               }

               Log.e(TAG, error);
               onCompletion(mediaPlayer);
               Toast.makeText(this, error, Toast.LENGTH_LONG).show();
               return true;
           }

           private void playerPrepare() {
               try {
                   SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(this);
                   Boolean hwDecode = prefs.getBoolean("hw_decode", false);

                   mediaPlayer = new MediaPlayer(this, hwDecode);
                   mediaController = new MediaController(this);

                   HashMap<String, String> options = new HashMap<>();
                   options.put("seekable", "0");
                   options.put("multiple_requests", "1");

                   String dataSource = Torrent2Http.URL + "files/" + file.replace(" ", "%20");
                   mediaPlayer.setDataSource(dataSource, options);
                   Log.d(TAG, "dataSource:" + dataSource);

                   mediaPlayer.setDisplay(surfaceView.getHolder());
                   mediaPlayer.setBufferSize(2048 * 1024);
                   //mediaPlayer.setPlaybackSpeed(1.0f);
                   mediaPlayer.setVideoChroma(MediaPlayer.VIDEOCHROMA_RGB565);

                   Log.d("Encoding:", subtitleEncoding);
                   if(!subtitleEncoding.equals("auto")) {
                       mediaPlayer.setTimedTextEncoding(subtitleEncoding);
                   }

                   mediaPlayer.prepareAsync();

                   mediaPlayer.setOnPreparedListener(this);
                   mediaPlayer.setOnCompletionListener(this);
                   mediaPlayer.setOnTimedTextListener(this);
                   mediaPlayer.setOnErrorListener(this);
                   mediaPlayer.setOnVideoSizeChangedListener(this);
               } catch(IOException e) {
                   e.printStackTrace();
               }
           }

           private void playerStart() {
               mediaController.setMediaPlayer(this);
               mediaController.setAnchorView(surfaceView);

               if(file != null && !file.isEmpty()) {
                   List<String> items = Arrays.asList(file.split("/"));
                   String fileName = items.get(items.size() - 1);
                   if(sub != null) {
                       List<String> items2 = Arrays.asList(sub.split("/"));
                       String subName = items2.get(items2.size() - 1);
                       fileName += System.getProperty("line.separator") + subName;
                   }
                   mediaController.setFileName(fileName);
               }

               handler.post(new Runnable() {
                   public void run() {
                       mediaController.setEnabled(true);
                       mediaController.setInstantSeeking(false);
                       mediaController.show(4000);
                   }
               });

               mediaPlayer.start();

               if(sub != null) {
                   mediaPlayer.addTimedTextSource(sub);
                   mediaPlayer.setTimedTextShown(true);
                   subtitlesEnabled = true;
               }

           }

           private void playerRelease() {
               if(mediaController != null) {
                   if(mediaController.isShowing()) {
                       mediaController.hide();
                   }
                   mediaController = null;
               }
               if(mediaPlayer != null) {
                   mediaPlayer.reset();
                   mediaPlayer.release();
                   mediaPlayer = null;
               }
           }

           private void subtitleTogglePrepare() {
               int subtitleButtonId = getResources().getIdentifier("mediacontroller_toggle_subtitles", "id", getPackageName());
               if(subtitleButtonId != 0) {
                   subtitleToggle = (ImageButton) mediaController.findViewById(subtitleButtonId);
                   subtitleToggle.setOnClickListener(new View.OnClickListener() {
                       @Override
                       public void onClick(View v) {
                           toggleSubtitles();
                       }
                   });
               }
           }

           private void setVideoSize() {
               ViewGroup.LayoutParams lp = surfaceView.getLayoutParams();
               DisplayMetrics disp = getResources().getDisplayMetrics();

               int windowWidth = disp.widthPixels, windowHeight = disp.heightPixels;
               float windowRatio = windowWidth / (float) windowHeight;
               float videoRatio = mediaPlayer.getVideoAspectRatio();

               int videoWidth = mediaPlayer.getVideoWidth();
               int videoHeight = mediaPlayer.getVideoHeight();

               lp.width = (windowRatio < videoRatio) ? windowWidth : (int) (videoRatio * windowHeight);
               lp.height = (windowRatio > videoRatio) ? windowHeight : (int) (windowWidth / videoRatio);

               surfaceView.setLayoutParams(lp);
               surfaceView.getHolder().setFixedSize(videoWidth, videoHeight);
           }

           private void toggleMediaControllerVisibility() {
               if(mediaController != null) {
                   if(subtitleToggle != null) {
                       if(subtitlesEnabled) {
                           subtitleToggle.setImageResource(R.drawable.mediacontroller_subtitles_off);
                       } else {
                           subtitleToggle.setImageResource(R.drawable.mediacontroller_subtitles_on);
                       }
                   }
                   if(mediaController.isShowing()) {
                       mediaController.hide();
                   } else {
                       mediaController.show();
                   }
               }
           }

           private void toggleSubtitles() {
               if(mediaPlayer != null) {
                   if(subtitlesEnabled) {
                       subtitleView.setVisibility(View.INVISIBLE);
                       subtitleToggle.setImageResource(R.drawable.mediacontroller_subtitles_on);
                       mediaPlayer.setTimedTextShown(false);
                       subtitlesEnabled = false;
                       Toast.makeText(this, getString(R.string.subtitles_disabled), Toast.LENGTH_SHORT).show();
                   } else {
                       subtitleView.setVisibility(View.VISIBLE);
                       subtitleToggle.setImageResource(R.drawable.mediacontroller_subtitles_off);
                       mediaPlayer.setTimedTextShown(true);
                       subtitlesEnabled = true;
                       Toast.makeText(this, getString(R.string.subtitles_enabled), Toast.LENGTH_SHORT).show();
                   }
               }
           }

}
