package com.bukanir.android.entities;

import android.os.Parcel;
import android.os.Parcelable;

public class Movie implements Comparable<Movie>, Parcelable {

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
    public String category;
    public String season;
    public String episode;
    public String quality;

    protected Movie(Parcel in) {
        id = in.readString();
        title = in.readString();
        year = in.readString();
        posterSmall = in.readString();
        posterMedium = in.readString();
        posterLarge = in.readString();
        posterXLarge = in.readString();
        size = in.readString();
        sizeHuman = in.readString();
        seeders = in.readString();
        magnetLink = in.readString();
        release = in.readString();
        category = in.readString();
        season = in.readString();
        episode = in.readString();
        quality = in.readString();
    }

    @Override
    public int describeContents() {
        return 0;
    }

    @Override
    public void writeToParcel(Parcel dest, int flags) {
        dest.writeString(id);
        dest.writeString(title);
        dest.writeString(year);
        dest.writeString(posterSmall);
        dest.writeString(posterMedium);
        dest.writeString(posterLarge);
        dest.writeString(posterXLarge);
        dest.writeString(size);
        dest.writeString(sizeHuman);
        dest.writeString(seeders);
        dest.writeString(magnetLink);
        dest.writeString(release);
        dest.writeString(category);
        dest.writeString(season);
        dest.writeString(episode);
        dest.writeString(quality);
    }

    @SuppressWarnings("unused")
    public static final Parcelable.Creator<Movie> CREATOR = new Parcelable.Creator<Movie>() {
        @Override
        public Movie createFromParcel(Parcel in) {
            return new Movie(in);
        }

        @Override
        public Movie[] newArray(int size) {
            return new Movie[size];
        }
    };

    @Override
    public int compareTo(Movie m) {
        return Integer.valueOf(m.seeders) - Integer.valueOf(this.seeders);
    }

    @Override
    public boolean equals(Object obj) {
        if(obj == null) return false;
        if(obj == this) return true;
        if(!(obj instanceof Movie)) return false;
        Movie movie = (Movie) obj;
        return movie.id.equals(this.id);
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
        result.append("  size: " + this.size + NL);
        result.append("  sizeHuman: " + this.sizeHuman + NL);
        result.append("  seeders: " + this.seeders + NL);
        result.append("  magnetLink: " + this.magnetLink + NL);
        result.append("  release: " + this.release + NL);
        result.append("  category: " + this.category + NL);
        result.append("  season: " + this.season + NL);
        result.append("  episode: " + this.episode + NL);
        result.append("  quality: " + this.quality + NL);
        result.append("}" + NL);

        return result.toString();
    }

}
