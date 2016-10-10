package com.bukanir.android.entities;

import java.io.Serializable;

public class TorrentFile implements Comparable<TorrentFile>, Serializable {

    public String name;
    public String save_path;
    public String url;
    public String size;
    public String offset;
    public String download;
    public String progress;

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