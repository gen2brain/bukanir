package com.bukanir.android.fragments;

import android.app.Activity;
import android.app.Dialog;
import android.app.ProgressDialog;
import android.content.DialogInterface;
import android.os.Bundle;
import android.support.v4.app.DialogFragment;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ProgressBar;

import com.bukanir.android.R;
import com.bukanir.android.activities.MoviesListActivity;
import com.bukanir.android.activities.SearchActivity;

public class ProgressFragment extends DialogFragment {

    private static final String TAG = "ProgressFragment";

    ProgressBar progressBar;
    ProgressDialog progressDialog;

    public static ProgressFragment newInstance(String message) {
        ProgressFragment fragment = new ProgressFragment();
        Bundle args = new Bundle();
        args.putString("message", message);
        fragment.setArguments(args);
        return fragment;
    }

    @Override
    public void onCreate(Bundle savedInstanceState) {
        Log.d(TAG, "onCreate");
        super.onCreate(savedInstanceState);
        progressDialog = new ProgressDialog(getActivity());
        setRetainInstance(true);
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        Log.d(TAG, "onCreateView");
        if(getShowsDialog()) {
            return super.onCreateView(inflater, container, savedInstanceState);
        } else {
            View view = inflater.inflate(R.layout.fragment_progress, container, false);
            progressBar = (ProgressBar) view.findViewById(R.id.progressBar);
            progressBar.setMax(100);
            progressBar.setProgress(0);
            return view;
        }
    }

    @Override
    public Dialog onCreateDialog(Bundle savedInstanceState) {
        Log.d(TAG, "onCreateDialog");
        progressDialog.setMessage(getArguments().getString("message"));
        progressDialog.setProgressStyle(ProgressDialog.STYLE_HORIZONTAL);
        progressDialog.setMax(100);
        progressDialog.setProgress(0);
        progressDialog.setCancelable(true);
        progressDialog.setCanceledOnTouchOutside(false);
        return progressDialog;
    }

    @Override
    public void onDestroy() {
        Log.d(TAG, "onDestroy");
        super.onDestroy();
        Activity activity = getActivity();
        if(activity instanceof MoviesListActivity) {
            ((MoviesListActivity) activity).cancelMovieTask();
        } else if(activity instanceof SearchActivity) {
            ((SearchActivity) activity).cancelSearchTask();
        }
    }

    public void setProgress(int progress) {
        if(progressDialog != null) {
            progressDialog.setProgress(progress);
        }
        if(progressBar != null) {
            progressBar.setProgress(progress);
        }
    }

}
