package com.bukanir.android.fragments;

import android.app.Activity;
import android.content.Intent;
import android.graphics.Bitmap;
import android.os.AsyncTask;
import android.os.Build;
import android.support.v4.app.Fragment;
import android.os.Bundle;
import android.support.v4.app.FragmentTransaction;
import android.util.Log;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.AdapterView;
import android.widget.BaseAdapter;
import android.widget.ImageView;
import android.widget.ListView;
import android.widget.ProgressBar;
import android.widget.TextView;
import android.widget.Toast;

import com.bukanir.android.application.Favorites;
import com.bukanir.android.clients.BukanirClient;
import com.bukanir.android.entities.Summary;
import com.bukanir.android.helpers.Connectivity;
import com.nostra13.universalimageloader.cache.disc.impl.UnlimitedDiskCache;
import com.nostra13.universalimageloader.core.DisplayImageOptions;
import com.nostra13.universalimageloader.core.ImageLoader;
import com.nostra13.universalimageloader.core.ImageLoaderConfiguration;
import com.nostra13.universalimageloader.core.display.FadeInBitmapDisplayer;

import com.bukanir.android.activities.MovieActivity;
import com.bukanir.android.entities.Movie;
import com.bukanir.android.R;
import com.bukanir.android.helpers.Utils;
import com.nostra13.universalimageloader.core.listener.ImageLoadingListener;
import com.nostra13.universalimageloader.core.listener.SimpleImageLoadingListener;

import java.io.File;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedList;
import java.util.List;

public class MoviesListFragment extends Fragment {

    public static final String TAG = "MoviesListFragment";

    private Movie movie;
    private Summary summary;
    ArrayList<Movie> movies;

    boolean twoPane;
    private Favorites favorites;

    DisplayImageOptions options;
    private MovieTask movieTask;
    private ProgressBar progressBar;
    private int selectedListItem;

    protected ImageLoader imageLoader = ImageLoader.getInstance();

    public static MoviesListFragment newInstance(ArrayList<Movie> movies, boolean mTwoPane) {
        MoviesListFragment fragment = new MoviesListFragment();
        Bundle args = new Bundle();
        args.putParcelableArrayList("movies", movies);
        args.putBoolean("twoPane", mTwoPane);
        fragment.setArguments(args);
        return fragment;
    }

    @Override
    public View onCreateView(LayoutInflater inflater, ViewGroup container, Bundle savedInstanceState) {
        Log.d(TAG, "onCreateView");

        favorites = new Favorites(getActivity());

        if(savedInstanceState != null) {
            movies = savedInstanceState.getParcelableArrayList("movies");
            selectedListItem = savedInstanceState.getInt("selectedListItem");
        } else {
            movies = getArguments().getParcelableArrayList("movies");
            selectedListItem = -1;
        }

        twoPane = getArguments().getBoolean("twoPane");

        View view = inflater.inflate(R.layout.fragment_movie_list, container, false);

        options = new DisplayImageOptions.Builder()
            .showImageOnLoading(R.drawable.ic_stub)
            .showImageForEmptyUri(R.drawable.ic_empty)
            .showImageOnFail(R.drawable.ic_error)
            .cacheInMemory(true)
            .cacheOnDisk(true)
            .build();

        if(!imageLoader.isInited()) {
            File imagesDir = new File(getActivity().getCacheDir().toString() + File.separator + "images");
            imagesDir.mkdirs();
            ImageLoaderConfiguration config = new
                ImageLoaderConfiguration.Builder(getActivity().getApplicationContext())
                .diskCache(new UnlimitedDiskCache(imagesDir))
                .defaultDisplayImageOptions(DisplayImageOptions.createSimple())
                .build();
            imageLoader.init(config);
        }

        return view;
    }

    @Override
    public void onViewCreated(View view, Bundle savedInstanceState) {
        super.onViewCreated(view, savedInstanceState);
        progressBar = (ProgressBar) view.getRootView().findViewById(R.id.progressbar);
        if(!movies.isEmpty()) {
            prepareListView(view);
        } else {
            String className = getActivity().getClass().getSimpleName();
            if(className.equals("FavoritesActivity")) {
                Toast.makeText(getActivity(), getString(R.string.favorites_empty), Toast.LENGTH_SHORT).show();
            }
        }
    }

    @Override
    public void onSaveInstanceState(Bundle outState) {
        Log.d(TAG, "onSaveInstanceState");
        super.onSaveInstanceState(outState);
        if(movies != null && !movies.isEmpty()) {
            outState.putParcelableArrayList("movies", movies);
        }
        if(selectedListItem != -1) {
            outState.putInt("selectedListItem", selectedListItem);
        }
    }

    @Override
    public void onPause() {
        super.onPause();
        cancelMovieTask();
    }

    @Override
    public void onResume() {
        Log.d(TAG, "onResume");
        super.onResume();
        if(progressBar != null) {
            progressBar.setVisibility(View.GONE);
        }
    }

    private void prepareListView(View view) {
        final ListView listView = (ListView) view.findViewById(R.id.movie_list);
        final ItemAdapter adapter = new ItemAdapter();

        listView.setAdapter(adapter);

        listView.setOnItemClickListener(new AdapterView.OnItemClickListener() {
            @Override
            public void onItemClick(AdapterView<?> parent, View view, int position, long id) {
                selectedListItem = position;
                movie = movies.get(selectedListItem);

                if (twoPane) {
                    startMovieTask();
                } else {
                    Intent intent = new Intent(getActivity(), MovieActivity.class);
                    intent.putExtra("movie", movies.get(position));
                    startActivity(intent);
                }
            }
        });

        listView.setOnItemLongClickListener(new AdapterView.OnItemLongClickListener() {
            @Override
            public boolean onItemLongClick(AdapterView<?> adapterView, View view, int position, long id) {
                String className = getActivity().getClass().getSimpleName();
                if(className.equals("MoviesListActivity") || className.equals("SearchActivity")) {
                    movie = movies.get(position);
                    favorites.addToFavorites(movie);
                    Toast.makeText(getActivity(), movie.title + getString(R.string.favorite_added), Toast.LENGTH_SHORT).show();
                } else if(className.equals("FavoritesActivity")) {
                    movie = movies.get(position);
                    favorites.removeFromFavorites(movie);
                    movies = favorites.getFavorites();
                    adapter.notifyDataSetChanged();
                    Toast.makeText(getActivity(), movie.title + getString(R.string.favorite_removed), Toast.LENGTH_SHORT).show();
                }
                return true;
            }
        });

        if(twoPane) {
            int v = 0;
            if(selectedListItem != -1) {
                v = selectedListItem;
            }
            listView.performItemClick(adapter.getView(v, null, null), v, adapter.getItemId(v));
        }
    }

    private void beginTransaction() {
        FragmentTransaction ft;
        Activity activity = getActivity();
        if(activity != null && activity.findViewById(R.id.movie_container) != null) {
            ft = getActivity().getSupportFragmentManager().beginTransaction();
            ft.replace(R.id.movie_container, MovieFragment.newInstance(movie, summary));
            ft.commitAllowingStateLoss();
        }
    }

    private void startMovieTask() {
        if(movieTask != null) {
            if(movieTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                movieTask.cancel(true);
            }
        }
        if(Connectivity.isConnected(getActivity())) {
            movieTask = new MovieTask();
            if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.HONEYCOMB) {
                movieTask.executeOnExecutor(AsyncTask.THREAD_POOL_EXECUTOR, movies.get(selectedListItem));
            } else {
                movieTask.execute(movies.get(selectedListItem));
            }
        } else {
            Toast.makeText(getActivity(), getString(R.string.network_not_available), Toast.LENGTH_LONG).show();
        }
    }

    public void cancelMovieTask() {
        if(movieTask != null) {
            if(movieTask.getStatus().equals(AsyncTask.Status.RUNNING)) {
                movieTask.cancel(true);
            }
        }
    }

    private class MovieTask extends AsyncTask<Movie, Void, Summary> {

        protected void onPreExecute() {
            super.onPreExecute();
            if(progressBar != null) {
                progressBar.setVisibility(View.VISIBLE);
            }
        }

        protected Summary doInBackground(Movie... params) {
            summary = BukanirClient.getSummary(
                    Integer.valueOf(movie.id),
                    Integer.valueOf(movie.category),
                    Integer.valueOf(movie.season),
                    Integer.valueOf(movie.episode));

            if(isCancelled()) {
                return null;
            }

            return summary;
        }

        protected void onPostExecute(Summary s) {
            if(progressBar != null) {
                progressBar.setVisibility(View.GONE);
            }
            if(s != null && movie != null) {
                beginTransaction();
            }
        }

    }

    class ItemAdapter extends BaseAdapter {

        private ImageLoadingListener animateFirstListener = new AnimateFirstDisplayListener();

        private class ViewHolder {
            public TextView title;
            public TextView year;
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
                holder.year = (TextView) view.findViewById(R.id.year);
                holder.image = (ImageView) view.findViewById(R.id.image);
                view.setTag(holder);
            } else {
                holder = (ViewHolder) view.getTag();
            }

            Movie m = movies.get(position);
            holder.title.setText(Utils.toTitleCase(m.title));

            if(m.category.equals("205") || m.category.equals("208")) {
                int season = Integer.valueOf(m.season);
                int episode = Integer.valueOf(m.episode);
                if(season != 0) {
                    String text = String.format("S%02dE%02d", season, episode);
                    if(m.category.equals("208") && m.quality != null && !m.quality.isEmpty()) {
                        text += String.format(" (%sp)", m.quality);;
                    }
                    holder.year.setText(text);
                }
            } else {
                String text = m.year;
                if(m.category.equals("207") && m.quality != null && !m.quality.isEmpty()) {
                    text += String.format(" (%sp)", m.quality);;
                }
                holder.year.setText(text);
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
