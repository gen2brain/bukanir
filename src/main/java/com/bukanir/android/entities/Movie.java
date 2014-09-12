package com.bukanir.android.entities;

import java.io.Serializable;

public class Movie implements Comparable<Movie>, Serializable {

    public String id;
    public String title;
    public String year;
    public String posterSmall;
    public String posterMedium;
    public String posterLarge;
    public String posterXLarge;
    public String size;
    public String sizeHuman;
    public String seeders;
    public String magnetLink;
    public String release;

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
	    result.append("  size: " + this.size + NL);
        result.append("  sizeHuman: " + this.sizeHuman + NL);
	    result.append("  seeders: " + this.seeders + NL);
	    result.append("  magnetLink: " + this.magnetLink + NL);
        result.append("  release: " + this.release + NL);
	    result.append("}" + NL);

	    return result.toString();
	}

	@Override
	public int compareTo(Movie m) {
		return Integer.valueOf(m.seeders) - Integer.valueOf(this.seeders);
	}

}
