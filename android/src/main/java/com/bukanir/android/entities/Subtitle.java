package com.bukanir.android.entities;

import java.io.Serializable;

public class Subtitle implements Comparable<Subtitle>, Serializable {

    public String id;
    public String title;
    public String year;
    public String release;
    public String downloadLink;
    public String score;

    @Override
    public int compareTo(Subtitle s) {
        if(s.score.isEmpty() || this.score.isEmpty()) {
            return 0;
        }
        return Double.compare(
                Double.parseDouble(s.score.replace(",", ".")),
                Double.parseDouble(this.score.replace(",", "."))
        );
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

}