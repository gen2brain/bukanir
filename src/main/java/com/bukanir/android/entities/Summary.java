package com.bukanir.android.entities;

import android.os.Parcel;
import android.os.Parcelable;

import java.util.ArrayList;
import java.util.List;

public class Summary implements Parcelable {

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

    protected Summary(Parcel in) {
        id = in.readString();
        video = in.readString();
        director = in.readString();
        rating = in.readString();
        tagline = in.readString();
        overview = in.readString();
        runtime = in.readString();
        imdbId = in.readString();
        if(in.readByte() == 0x01) {
            cast = new ArrayList<>();
            in.readList(cast, String.class.getClassLoader());
        } else {
            cast = null;
        }
        if(in.readByte() == 0x01) {
            genre = new ArrayList<>();
            in.readList(genre, String.class.getClassLoader());
        } else {
            genre = null;
        }
    }

    @Override
    public int describeContents() {
        return 0;
    }

    @Override
    public void writeToParcel(Parcel dest, int flags) {
        dest.writeString(id);
        dest.writeString(video);
        dest.writeString(director);
        dest.writeString(rating);
        dest.writeString(tagline);
        dest.writeString(overview);
        dest.writeString(runtime);
        dest.writeString(imdbId);
        if(cast == null) {
            dest.writeByte((byte) (0x00));
        } else {
            dest.writeByte((byte) (0x01));
            dest.writeList(cast);
        }
        if(genre == null) {
            dest.writeByte((byte) (0x00));
        } else {
            dest.writeByte((byte) (0x01));
            dest.writeList(genre);
        }
    }

    @SuppressWarnings("unused")
    public static final Parcelable.Creator<Summary> CREATOR = new Parcelable.Creator<Summary>() {
        @Override
        public Summary createFromParcel(Parcel in) {
            return new Summary(in);
        }

        @Override
        public Summary[] newArray(int size) {
            return new Summary[size];
        }
    };

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