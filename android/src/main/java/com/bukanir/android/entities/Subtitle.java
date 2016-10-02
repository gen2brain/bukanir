package com.bukanir.android.entities;

import android.os.Parcel;
import android.os.Parcelable;

public class Subtitle implements Comparable<Subtitle>, Parcelable {

    public String id;
    public String title;
    public String year;
    public String release;
    public String downloadLink;
    public String score;

    protected Subtitle(Parcel in) {
        id = in.readString();
        title = in.readString();
        year = in.readString();
        release = in.readString();
        downloadLink = in.readString();
        score = in.readString();
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
        dest.writeString(release);
        dest.writeString(downloadLink);
        dest.writeString(score);
    }

    @SuppressWarnings("unused")
    public static final Parcelable.Creator<Subtitle> CREATOR = new Parcelable.Creator<Subtitle>() {
        @Override
        public Subtitle createFromParcel(Parcel in) {
            return new Subtitle(in);
        }

        @Override
        public Subtitle[] newArray(int size) {
            return new Subtitle[size];
        }
    };

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