package com.bukanir.android.application;

import android.content.Context;
import android.content.SharedPreferences;
import android.preference.PreferenceManager;

public class Settings {

    private SharedPreferences preferences;

    public Settings(Context context) {
        Context context1 = context.getApplicationContext();
        preferences = PreferenceManager.getDefaultSharedPreferences(context1);
    }

    public int listCount() {
        return Integer.valueOf(preferences.getString("list_count", "30"));
    }

    public int cacheDays() {
        return Integer.valueOf(preferences.getString("cache_days", "7"));
    }

    public boolean eulaAccepted() {
        return preferences.getBoolean("eula_accepted", false);
    }

    public boolean hwDecode() {
        return preferences.getBoolean("hw_decode", false);
    }

    public boolean openSLES() {
        return preferences.getBoolean("open_sles", false);
    }

    public boolean seek() {
        return preferences.getBoolean("seek", true);
    }

    public String pixelFormat() {
        return preferences.getString("pixel_format", "");
    }

    public boolean wifiHigh() {
        return preferences.getBoolean("wifi_high", false);
    }

    public boolean subtitles() {
        return preferences.getBoolean("subtitles", true);
    }

    public String subtitleLanguage() {
        return preferences.getString("sub_lang", "English");
    }

    public String subtitleSize() {
        return preferences.getString("sub_size", "14");
    }

    public boolean keepFiles() {
        return preferences.getBoolean("keep_files", false);
    }

    public boolean encryption() {
        return preferences.getBoolean("encryption", true);
    }

    public String downloadRate() {
        return preferences.getString("download_rate", "-1");
    }

    public String uploadRate() {
        return preferences.getString("upload_rate", "-1");
    }

    public String listenPort() {
        return String.valueOf(preferences.getInt("listen_port", 6881));
    }

    public String tpbHost() {
        return String.valueOf(preferences.getString("tpb_host", "thepiratebay.org"));
    }

    public String eztvHost() {
        return String.valueOf(preferences.getString("eztv_host", "eztv.ag"));
    }

}
