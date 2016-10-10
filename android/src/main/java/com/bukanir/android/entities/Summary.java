package com.bukanir.android.entities;

import java.io.Serializable;
import java.util.List;

public class Summary implements Serializable {

    public String id;
    public String video;
    public String director;
    public String rating;
    public String tagline;
    public String overview;
    public String runtime;
    public String imdbId;

    public List<String> cast;
    public List<String> genre;

    @Override
    public String toString() {
        StringBuilder result = new StringBuilder();
        String NL = System.getProperty("line.separator");

        result.append(((Object)this).getClass().getName() + " {" + NL);
        result.append("  id: " + this.id + NL);
        result.append("  video: " + this.video + NL);
        result.append("  director: " + this.director + NL);
        result.append("  rating: " + this.rating + NL);
        result.append("  tagline: " + this.tagline + NL);
        result.append("  overview: " + this.overview + NL);
        result.append("  runtime: " + this.runtime + NL);
        result.append("  imdbId: " + this.imdbId + NL);
        if(this.cast != null && !this.cast.isEmpty()) {
            result.append("  cast: " + android.text.TextUtils.join(", ", this.cast) + NL);
        }
        if(this.genre != null && !this.genre.isEmpty()) {
            result.append("  genre: " + android.text.TextUtils.join(", ", this.genre) + NL);
        }
        result.append("}" + NL);

        return result.toString();
    }

}