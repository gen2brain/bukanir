package com.bukanir.android.entities;

public class TorrentFile implements Comparable<TorrentFile> {

    public String name;
    public String size;
    public String offset;
    public String total_pieces;
    public String complete_pieces;

    @Override
    public String toString() {
        StringBuilder result = new StringBuilder();
        String NL = System.getProperty("line.separator");

        result.append(((Object)this).getClass().getName() + " {" + NL);
        result.append("  name: " + this.name + NL);
        result.append("  size: " + this.size + NL);
        result.append("  offset: " + this.offset + NL);
        result.append("  total_pieces: " + this.total_pieces + NL);
        result.append("  complete_pieces: " + this.complete_pieces + NL);
        result.append("}" + NL);

        return result.toString();
    }

    @Override
    public int compareTo(TorrentFile f) {
        return (int) (Long.valueOf(f.size) - Long.valueOf(this.size));
    }

}
