package com.bukanir.android.fragments;

import android.content.Intent;
import android.graphics.Bitmap;
import android.os.AsyncTask;
import android.support.v4.app.Fragment;
import android.os.Bundle;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.AdapterView;
import android.widget.BaseAdapter;
import android.widget.ImageView;
import android.widget.ListAdapter;
import android.widget.ListView;
import android.widget.ProgressBar;
import android.widget.TextView;

import com.bukanir.android.BukanirClient;
import com.bukanir.android.entities.Summary;
import com.nostra13.universalimageloader.core.DisplayImageOptions;
import com.nostra13.universalimageloader.core.ImageLoader;
import com.nostra13.universalimageloader.core.ImageLoaderConfiguration;
import com.nostra13.universalimageloader.core.assist.ImageLoadingListener;
import com.nostra13.universalimageloader.core.assist.SimpleImageLoadingListener;
import com.nostra13.universalimageloader.core.display.FadeInBitmapDisplayer;
import com.nostra13.universalimageloader.core.display.SimpleBitmapDisplayer;

import com.bukanir.android.activities.MovieActivity;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.R;
import com.bukanir.android.utils.Utils;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedList;
import java.util.List;

public class SearchFragment extends Fragment {

    public static final String TAG = "SearchFragment";

    boolean twoPane;
    ArrayList<Movie> movies;

    private ProgressBar progressBar;

    DisplayImageOptions options;
    protected ImageLoader imageLoader = ImageLoader.getInstance();

    public static SearchFragment newInstance(ArrayList<Movie> movies, boolean mTwoPane) {
        SearchFragment fragment = new SearchFragment();
        Bundle args = new Bundle();
        args.putSerializable("search", movies);
        args.putBoolean("twoPane", mTwoPane);
        fragment.setArguments(args);
        return fragment;
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        Log.d(TAG, "onCreateView");

        if(savedInstanceState != null) {
            movies = (ArrayList<Movie>) savedInstanceState.getSerializable("search");
        } else {
            movies = (ArrayList<Movie>) getArguments().getSerializable("search");
        }

        twoPane = getArguments().getBoolean("twoPane");

        View rootView = inflater.inflate(R.layout.fragment_movie_list, container, false);

        options = new DisplayImageOptions.Builder()
                .showImageOnLoading(R.drawable.ic_stub)
                .showImageForEmptyUri(R.drawable.ic_empty)
                .showImageOnFail(R.drawable.ic_error)
                .cacheInMemory(true)
                .cacheOnDisc(true)
                .considerExifParams(true)
                .displayer(new SimpleBitmapDisplayer())
                .build();

        if(!imageLoader.isInited()) {
            imageLoader.init(ImageLoaderConfiguration.createDefault(getActivity()));
        }

        prepareListView(rootView);

        return rootView;
    }

    @Override
    public void onViewCreated(View view, Bundle savedInstanceState) {
        super.onViewCreated(view, savedInstanceState);
        View v = getView().getRootView();
        if(v != null) {
            progressBar = (ProgressBar) v.findViewById(R.id.progressbar);
        }
    }

    @Override
    public void onSaveInstanceState(Bundle outState) {
        Log.d(TAG, "onSaveInstanceState");
        super.onSaveInstanceState(outState);
        outState.putSerializable("search", movies);
    }

    private void prepareListView(View view) {
        ListView listView = (ListView) view.findViewById(R.id.movie_list);
        ListAdapter adapter = new ItemAdapter();
        listView.setAdapter(adapter);

        listView.setOnItemClickListener(new AdapterView.OnItemClickListener() {
            @Override
            public void onItemClick(AdapterView<?> parent, View view, int position, long id) {
                startMovieActivity(position);
            }
        });

        if(twoPane) {
            listView.performItemClick(listView.getChildAt(0), 0, adapter.getItemId(0));
        }
    }

    private void beginTransaction(Movie movie, Summary summary) {
        getActivity().getSupportFragmentManager().beginTransaction()
                .replace(R.id.movie_container, MovieFragment.newInstance(movie, summary))
                .commit();
    }

    private void startMovieActivity(int position) {
        if(twoPane) {
            new MovieTask().execute(movies.get(position));
        } else {
            Intent intent = new Intent(getActivity(), MovieActivity.class);
            intent.putExtra("movie", movies.get(position));
            startActivity(intent);
        }
    }

    private class MovieTask extends AsyncTask<Movie, Void, Summary> {

        private Movie movie;

        protected void onPreExecute() {
            super.onPreExecute();
            if(progressBar != null) {
                progressBar.setVisibility(View.VISIBLE);
            }
        }

        protected Summary doInBackground(Movie... params) {
            movie = params[0];

            if(isCancelled()) {
                return null;
            }

            Summary summary = BukanirClient.getSummary(Integer.valueOf(movie.id), Integer.valueOf(movie.category), Integer.valueOf(movie.season));
            return summary;
        }

        protected void onPostExecute(Summary summary) {
            if(progressBar != null) {
                progressBar.setVisibility(View.GONE);
            }
            if(summary != null && movie != null) {
                beginTransaction(movie, summary);
            }
        }

    }

    class ItemAdapter extends BaseAdapter {

        private ImageLoadingListener animateFirstListener = new AnimateFirstDisplayListener();

        private class ViewHolder {
            public TextView title;
            public TextView subtitle;
            public ImageView image;
        }

        @Override
        public int getCount() {
            if(movies != null) {
                return movies.size();
            } else {
                return 0;
            }
        }

        @Override
        public Object getItem(int position) {
            return movies.get(position);
        }

        @Override
        public long getItemId(int position) {
            return position;
        }

        @Override
        public View getView(final int position, View convertView, ViewGroup parent) {
            View view = convertView;
            final ViewHolder holder;
            if (convertView == null) {
                LayoutInflater inflater = getLayoutInflater(null);
                view = inflater.inflate(R.layout.item_list_image, parent, false);

                holder = new ViewHolder();
                holder.title = (TextView) view.findViewById(R.id.title);
                holder.subtitle = (TextView) view.findViewById(R.id.subtitle);
                holder.image = (ImageView) view.findViewById(R.id.image);
                view.setTag(holder);
            } else {
                holder = (ViewHolder) view.getTag();
            }

            String title = Utils.toTitleCase(movies.get(position).title);
            holder.title.setText(title);

            if(movies.get(position).category.equals("205")) {
                int season = Integer.valueOf(movies.get(position).season);
                int episode = Integer.valueOf(movies.get(position).episode);
                if(season != 0) {
                    holder.subtitle.setText(String.format("S%02dE%02d", season, episode));
                }
            } else {
                holder.subtitle.setText(movies.get(position).year);
            }

            imageLoader.displayImage(movies.get(position).posterSmall, holder.image, options, animateFirstListener);

            return view;
        }
    }

    private static class AnimateFirstDisplayListener extends SimpleImageLoadingListener {

        static final List<String> displayedImages = Collections.synchronizedList(new LinkedList<String>());

        @Override
        public void onLoadingComplete(String imageUri, View view, Bitmap loadedImage) {
            if(loadedImage != null) {
                ImageView imageView = (ImageView) view;
                boolean firstDisplay = !displayedImages.contains(imageUri);
                if(firstDisplay) {
                    FadeInBitmapDisplayer.animate(imageView, 500);
                    displayedImages.add(imageUri);
                }
            }
        }
    }

}
