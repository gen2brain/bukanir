package com.bukanir.android.entities;

import java.io.Serializable;
import java.util.List;

public class Subtitle implements Comparable<Subtitle>, Serializable {
	
	private static final long serialVersionUID = 6325576090541429026L;

    public String id;
	public String title;
	public String year;
	public String release;
	public String downloadLink;
	public String score;
	
	public Subtitle(List<String> res) {
        this.id = res.get(0);
		this.title = res.get(1).toLowerCase();
		this.year = res.get(2);
		this.release = res.get(3);
		this.downloadLink = res.get(4);
		this.score = res.get(5);
	}
	
	@Override
	public String toString() {
	    StringBuilder result = new StringBuilder();
	    String NL = System.getProperty("line.separator");
	
	    result.append(((Object)this).getClass().getName() + " {" + NL);
	    result.append("  id: " + this.id + NL);
        result.append("  title: " + this.title + NL);
	    result.append("  year: " + this.year + NL);
	    result.append("  release: " + this.release + NL);
	    result.append("  downloadLink: " + this.downloadLink + NL);
	    result.append("  score: " + this.score + NL);
	    result.append("}" + NL);
	
	    return result.toString();
	}

	@Override
	public int compareTo(Subtitle s) {
		if(s.score.isEmpty() || this.score.isEmpty()) {
			return 0;
		}
		return Double.compare(Double.parseDouble(s.score), Double.parseDouble(this.score));
	}	

}
