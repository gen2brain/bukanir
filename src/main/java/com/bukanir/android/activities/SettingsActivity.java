package com.bukanir.android.activities;

import android.annotation.TargetApi;
import android.content.Context;
import android.content.res.Configuration;
import android.os.Build;
import android.os.Bundle;
import android.preference.ListPreference;
import android.preference.Preference;
import android.preference.PreferenceActivity;
import android.preference.PreferenceCategory;
import android.preference.PreferenceFragment;
import android.preference.PreferenceManager;
import android.support.v7.widget.Toolbar;
import android.util.AttributeSet;
import android.util.Log;
import android.util.TypedValue;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.LinearLayout;
import android.widget.ListView;
import android.support.v7.widget.AppCompatCheckBox;
import android.support.v7.widget.AppCompatCheckedTextView;
import android.support.v7.widget.AppCompatEditText;
import android.support.v7.widget.AppCompatRadioButton;
import android.support.v7.widget.AppCompatSpinner;
import android.widget.TextView;

import com.bukanir.android.R;
import com.quietlycoding.android.picker.NumberPickerPreference;

import java.util.List;

public class SettingsActivity extends PreferenceActivity {

    private static final String TAG = "SettingsActivity";

    @Override
    public View onCreateView(String name, Context context, AttributeSet attrs) {
        final View result = super.onCreateView(name, context, attrs);
        if(result != null) {
            return result;
        }

        if(Build.VERSION.SDK_INT < Build.VERSION_CODES.LOLLIPOP) {
            switch (name) {
                case "EditText":
                    return new AppCompatEditText(this, attrs);
                case "Spinner":
                    return new AppCompatSpinner(this, attrs);
                case "CheckBox":
                    return new AppCompatCheckBox(this, attrs);
                case "RadioButton":
                    return new AppCompatRadioButton(this, attrs);
                case "CheckedTextView":
                    return new AppCompatCheckedTextView(this, attrs);
            }
        }

        return null;
    }

    @Override
    protected void onPostCreate(Bundle savedInstanceState) {
        super.onPostCreate(savedInstanceState);

        Toolbar toolbar;

        if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.ICE_CREAM_SANDWICH) {
            LinearLayout root = (LinearLayout) findViewById(android.R.id.list).getParent().getParent().getParent();
            toolbar = (Toolbar) LayoutInflater.from(this).inflate(R.layout.toolbar_settings, root, false);
            toolbar.setLogo(R.drawable.ic_launcher);
            root.addView(toolbar, 0);
        } else {
            ViewGroup root = (ViewGroup) findViewById(android.R.id.content);
            ListView content = (ListView) root.getChildAt(0);

            root.removeAllViews();

            toolbar = (Toolbar) LayoutInflater.from(this).inflate(R.layout.toolbar_settings, root, false);
            toolbar.setLogo(R.drawable.ic_launcher);

            int height;
            TypedValue tv = new TypedValue();
            if(getTheme().resolveAttribute(R.attr.actionBarSize, tv, true)) {
                height = TypedValue.complexToDimensionPixelSize(tv.data, getResources().getDisplayMetrics());
            } else {
                height = toolbar.getHeight();
            }

            content.setPadding(0, height, 0, 0);

            root.addView(content);
            root.addView(toolbar);
        }

        toolbar.setNavigationOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                finish();
            }
        });

        setupSimplePreferencesScreen();
    }

    private void setupSimplePreferencesScreen() {
        if (!isSimplePreferences(this)) {
            return;
        }

        addPreferencesFromResource(R.xml.pref_general);

        PreferenceCategory headerPlayer;
        headerPlayer = new PreferenceCategory(this);
        headerPlayer.setTitle(R.string.pref_header_player);
        getPreferenceScreen().addPreference(headerPlayer);
        addPreferencesFromResource(R.xml.pref_player);

        PreferenceCategory headerSubtitles;
        headerSubtitles = new PreferenceCategory(this);
        headerSubtitles.setTitle(R.string.pref_header_subtitles);
        getPreferenceScreen().addPreference(headerSubtitles);
        addPreferencesFromResource(R.xml.pref_subtitles);

        PreferenceCategory headerTorrent;
        headerTorrent = new PreferenceCategory(this);
        headerTorrent.setTitle(R.string.pref_header_torrent);
        getPreferenceScreen().addPreference(headerTorrent);
        addPreferencesFromResource(R.xml.pref_torrents);

        bindPreferenceSummaryToValue(findPreference("list_count"));
        bindPreferenceSummaryToValue(findPreference("cache_days"));
        bindPreferenceSummaryToValue(findPreference("pixel_format"));
        bindPreferenceSummaryToValue(findPreference("sub_lang"));
        bindPreferenceSummaryToValue(findPreference("sub_size"));
        bindPreferenceSummaryToValue(findPreference("download_rate"));
        bindPreferenceSummaryToValue(findPreference("upload_rate"));
        bindPreferenceSummaryToIntValue(findPreference("listen_port"));
    }

    @Override
    public boolean onIsMultiPane() {
        return isXLargeTablet(this) && !isSimplePreferences(this);
    }

    private static boolean isXLargeTablet(Context context) {
        return (context.getResources().getConfiguration().screenLayout
                & Configuration.SCREENLAYOUT_SIZE_MASK) >= Configuration.SCREENLAYOUT_SIZE_LARGE;
    }

    private static boolean isSimplePreferences(Context context) {
        return Build.VERSION.SDK_INT < Build.VERSION_CODES.HONEYCOMB
                || !isXLargeTablet(context);
    }

    @Override
    @TargetApi(Build.VERSION_CODES.HONEYCOMB)
    public void onBuildHeaders(List<Header> target) {
        if(!isSimplePreferences(this)) {
            loadHeadersFromResource(R.xml.pref_headers, target);
        }
    }

    @Override
    protected boolean isValidFragment(String fragmentName) {
        return true;
    }

    private static Preference.OnPreferenceChangeListener sBindPreferenceSummaryToValueListener = new Preference.OnPreferenceChangeListener() {
        @Override
        public boolean onPreferenceChange(Preference preference, Object value) {
            Log.d(TAG, "onPreferenceChange");
            String stringValue = value.toString();

            if(preference instanceof ListPreference) {
                ListPreference listPreference = (ListPreference) preference;
                int index = listPreference.findIndexOfValue(stringValue);
                preference.setSummary(index >= 0 ? listPreference.getEntries()[index] : null);
            } else if(preference instanceof NumberPickerPreference) {
                NumberPickerPreference numberPreference = (NumberPickerPreference) preference;
                preference.setSummary(String.valueOf(numberPreference.getValue()));
            } else {
                preference.setSummary(stringValue);
            }
            return true;
        }
    };

    private static void bindPreferenceSummaryToValue(Preference preference) {
        preference.setOnPreferenceChangeListener(sBindPreferenceSummaryToValueListener);

        sBindPreferenceSummaryToValueListener.onPreferenceChange(preference,
                PreferenceManager.getDefaultSharedPreferences(preference.getContext()).getString(preference.getKey(), ""));
    }

    private static void bindPreferenceSummaryToIntValue(Preference preference) {
        preference.setOnPreferenceChangeListener(sBindPreferenceSummaryToValueListener);

        sBindPreferenceSummaryToValueListener.onPreferenceChange(preference,
                String.valueOf(PreferenceManager.getDefaultSharedPreferences(preference.getContext()).getInt(preference.getKey(), 0)));
    }

    @TargetApi(Build.VERSION_CODES.HONEYCOMB)
    public static class GeneralPreferenceFragment extends PreferenceFragment {
        @Override
        public void onCreate(Bundle savedInstanceState) {
            super.onCreate(savedInstanceState);
            addPreferencesFromResource(R.xml.pref_general);

            bindPreferenceSummaryToValue(findPreference("list_count"));
            bindPreferenceSummaryToValue(findPreference("cache_days"));
        }
    }

    @TargetApi(Build.VERSION_CODES.HONEYCOMB)
    public static class PlayerPreferenceFragment extends PreferenceFragment {
        @Override
        public void onCreate(Bundle savedInstanceState) {
            super.onCreate(savedInstanceState);
            addPreferencesFromResource(R.xml.pref_player);
            bindPreferenceSummaryToValue(findPreference("pixel_format"));
        }
    }

    @TargetApi(Build.VERSION_CODES.HONEYCOMB)
    public static class TorrentsPreferenceFragment extends PreferenceFragment {
        @Override
        public void onCreate(Bundle savedInstanceState) {
            super.onCreate(savedInstanceState);
            addPreferencesFromResource(R.xml.pref_torrents);

            bindPreferenceSummaryToValue(findPreference("download_rate"));
            bindPreferenceSummaryToValue(findPreference("upload_rate"));
            bindPreferenceSummaryToIntValue(findPreference("listen_port"));
        }
    }

    @TargetApi(Build.VERSION_CODES.HONEYCOMB)
    public static class SubtitlesPreferenceFragment extends PreferenceFragment {
        @Override
        public void onCreate(Bundle savedInstanceState) {
            super.onCreate(savedInstanceState);
            addPreferencesFromResource(R.xml.pref_subtitles);

            bindPreferenceSummaryToValue(findPreference("sub_lang"));
            bindPreferenceSummaryToValue(findPreference("sub_size"));
        }
    }

}
