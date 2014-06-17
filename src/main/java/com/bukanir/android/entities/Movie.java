package com.bukanir.android.entities;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

import com.bukanir.android.scrapers.TheMovieDb;

public class Movie implements Comparable<Movie>, Serializable {

    private static final long serialVersionUID = -5727210591367483499L;

    public String id;
    public String title;
    public String year;
    public String posterSmall;
    public String posterMedium;
    public String posterLarge;
    public String posterXLarge;
    public String cast;
    public String rating;
    public String tagline;
    public String overview;
    public String runtime;
    public String release;
    public String size;
    public String seeders;
    public String magnetLink;

    public Movie(List<String> res) {
        this.id = res.get(0);
        this.title = res.get(1);
        this.year = res.get(2);
        this.posterSmall = res.get(3);
        this.posterMedium = res.get(4);
        this.posterLarge = res.get(5);
        this.posterXLarge = res.get(6);
        this.rating = res.get(7);
        this.release = res.get(8);
        this.size = res.get(9);
        this.seeders = res.get(10);
        this.magnetLink = res.get(11);
    }

    public void getSummary() {
        TheMovieDb tmdb = new TheMovieDb();
        ArrayList<String> summary = tmdb.getSummary(this.id);
        this.tagline = summary.get(3);
        this.overview = summary.get(4);
        this.runtime = summary.get(6);
        this.cast = summary.get(11);
    }

	@Override
	public String toString() {
	    StringBuilder result = new StringBuilder();
	    String NL = System.getProperty("line.separator");

	    result.append(((Object)this).getClass().getName() + " {" + NL);
	    result.append("  title: " + this.title + NL);
	    result.append("  id: " + this.id + NL);
	    result.append("  year: " + this.year + NL);
	    result.append("  posterSmall: " + this.posterSmall + NL);
        result.append("  posterMedium: " + this.posterMedium + NL);
        result.append("  posterLarge: " + this.posterLarge + NL);
	    result.append("  posterXLarge: " + this.posterXLarge + NL);
        result.append("  cast: " + this.cast + NL);
	    result.append("  rating: " + this.rating + NL);
	    result.append("  tagline: " + this.tagline + NL);
	    result.append("  overview: " + this.overview + NL);
	    result.append("  runtime: " + this.runtime + NL);
	    result.append("  release: " + this.release + NL);
	    result.append("  size: " + this.size + NL);
	    result.append("  seeders: " + this.seeders + NL);
	    result.append("  magnetLink: " + this.magnetLink + NL);
	    result.append("}" + NL);

	    return result.toString();
	}

	@Override
	public int compareTo(Movie m) {
		return Integer.valueOf(m.seeders) - Integer.valueOf(this.seeders);
	}

}
