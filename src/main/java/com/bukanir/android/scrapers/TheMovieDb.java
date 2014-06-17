package com.bukanir.android.scrapers;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import com.bukanir.android.utils.Utils;
import com.uwetrottmann.tmdb.Tmdb;
import com.uwetrottmann.tmdb.entities.AppendToResponse;
import com.uwetrottmann.tmdb.entities.Configuration;
import com.uwetrottmann.tmdb.entities.Credits;
import com.uwetrottmann.tmdb.entities.Movie;
import com.uwetrottmann.tmdb.entities.ResultsPage;
import com.uwetrottmann.tmdb.enumerations.AppendToResponseItem;

public class TheMovieDb {

    protected static final String API_KEY = "YOUR_API_KEY";

    private static final boolean DEBUG = false;

    private final Tmdb tmdb = new Tmdb();
    private final Configuration cfg;

    public TheMovieDb() {
    	tmdb.setApiKey(API_KEY);
    	tmdb.setIsDebug(DEBUG);
    	cfg = tmdb.configurationService().configuration();
    }
    
	public ArrayList<String> search(String query, String torrentYear) throws Exception {
		ResultsPage movieResults = tmdb.searchService().movie(query);
		
		if(movieResults.results.isEmpty()) {
			return null;
		}
		
		String year = "";
		Movie movie = null;
		for(Movie result : movieResults.results) {
			if(result.release_date != null && !torrentYear.isEmpty()) {
				int torrentYearInt = Integer.parseInt(torrentYear);
				year = String.valueOf(result.release_date.getYear() + 1900);
				if(torrentYear.equals(year) || String.valueOf(torrentYearInt+1).equals(year) || String.valueOf(torrentYearInt-1).equals(year)) {
					movie = result;
					break;
				}
			}
		}
		
		if(movie == null) {
			movie = movieResults.results.get(0);
		}
		
		String id = String.valueOf(movie.id);
		String title = movie.title.toLowerCase();
		String posterSmall = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(0));
		String posterMedium = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(1));
        String posterLarge = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(2));
        String posterXLarge = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(3));
		String rating = String.valueOf(movie.vote_average);
		
		ArrayList<String> results = new ArrayList<String>(Arrays.asList(id, title, year, posterSmall, posterMedium, posterLarge, posterXLarge, rating));
		return results;
	}
	
	public ArrayList<String> getSummary(String tmdbId) {
		Movie movie = tmdb.moviesService().summary(Integer.valueOf(tmdbId), null, new AppendToResponse(AppendToResponseItem.CREDITS));
		String id = String.valueOf(movie.id);
		String title = movie.title;
		String year = String.valueOf(movie.release_date.getYear() + 1900);
		String tagline = movie.tagline;
		String overview = movie.overview;
		String rating = String.valueOf(movie.vote_average);
		String runtime = String.valueOf(movie.runtime);
		String posterSmall = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(0));
        String posterMedium = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(1));
        String posterLarge = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(2));
        String posterXLarge = getPosterUrl(movie.poster_path, cfg.images.poster_sizes.get(3));

        List<String> castList = new ArrayList<>();
        List<Credits.CastMember> credits = movie.credits.cast;
        if(!credits.isEmpty() && credits.size() >= 3) {
            for (Credits.CastMember cm : movie.credits.cast.subList(0, 3)) {
                castList.add(cm.name);
            }
        }
        String cast = Utils.join(castList, ", ");

		ArrayList<String> results = new ArrayList<String>(Arrays.asList(id, title, year, tagline, overview, rating, runtime, posterSmall, posterMedium, posterLarge, posterXLarge, cast));
		return results;
	}
	
	private String getPosterUrl(String posterPath, String size) {
		return cfg.images.base_url + size + posterPath;
	}

}
