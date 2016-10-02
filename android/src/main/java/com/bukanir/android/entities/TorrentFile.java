package com.bukanir.android.entities;

import android.os.Parcel;
import android.os.Parcelable;

public class TorrentFile implements Comparable<TorrentFile>, Parcelable {

    public String name;
    public String save_path;
    public String url;
    public String size;
    public String offset;
    public String download;
    public String progress;

    protected TorrentFile(Parcel in) {
        name = in.readString();
        save_path = in.readString();
        url = in.readString();
        size = in.readString();
        offset = in.readString();
        download = in.readString();
        progress = in.readString();
    }

    @Override
    public int describeContents() {
        return 0;
    }

    @Override
    public void writeToParcel(Parcel dest, int flags) {
        dest.writeString(name);
        dest.writeString(save_path);
        dest.writeString(url);
        dest.writeString(size);
        dest.writeString(offset);
        dest.writeString(download);
        dest.writeString(progress);
    }

    @SuppressWarnings("unused")
    public static final Parcelable.Creator<TorrentFile> CREATOR = new Parcelable.Creator<TorrentFile>() {
        @Override
        public TorrentFile createFromParcel(Parcel in) {
            return new TorrentFile(in);
        }

        @Override
        public TorrentFile[] newArray(int size) {
            return new TorrentFile[size];
        }
    };

    @Override
    public int compareTo(TorrentFile f) {
        return (int) (Long.valueOf(f.size) - Long.valueOf(this.size));
    }

    @Override
    public String toString() {
        StringBuilder result = new StringBuilder();
        String NL = System.getProperty("line.separator");

        result.append(((Object)this).getClass().getName() + " {" + NL);
        result.append("  name: " + this.name + NL);
        result.append("  save_path: " + this.save_path + NL);
        result.append("  url: " + this.url + NL);
        result.append("  size: " + this.size + NL);
        result.append("  offset: " + this.offset + NL);
        result.append("  download: " + this.download + NL);
        result.append("  progress: " + this.progress + NL);
        result.append("}" + NL);

        return result.toString();
    }

}