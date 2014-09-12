package com.bukanir.android.entities;

import java.io.Serializable;

public class Summary implements Serializable {

    public String id;
    public String cast;
    public String rating;
    public String tagline;
    public String overview;
    public String runtime;

	@Override
	public String toString() {
	    StringBuilder result = new StringBuilder();
	    String NL = System.getProperty("line.separator");

	    result.append(((Object)this).getClass().getName() + " {" + NL);
	    result.append("  id: " + this.id + NL);
        result.append("  cast: " + this.cast + NL);
	    result.append("  rating: " + this.rating + NL);
	    result.append("  tagline: " + this.tagline + NL);
	    result.append("  overview: " + this.overview + NL);
	    result.append("  runtime: " + this.runtime + NL);
	    result.append("}" + NL);

	    return result.toString();
	}

}
