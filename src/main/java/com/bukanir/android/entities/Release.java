package com.bukanir.android.entities;

public class Release implements Comparable<Release> {

	public String name;
	public String score;
	
	public Release(String name, String score) {
		this.name = name;
		this.score = score;
	}
	
	@Override
	public String toString() {
	    StringBuilder result = new StringBuilder();
	    String NL = System.getProperty("line.separator");
	
	    result.append(this.getClass().getName() + " {" + NL);
	    result.append("  name: " + this.name + NL);
	    result.append("  score: " + this.score + NL);
	    result.append("}" + NL);
	
	    return result.toString();
	}

	@Override
	public int compareTo(Release s) {
		if(s.score.isEmpty() || this.score.isEmpty()) {
			return 0;
		}
		return Double.compare(Double.parseDouble(s.score), Double.parseDouble(this.score));
	}	

}
