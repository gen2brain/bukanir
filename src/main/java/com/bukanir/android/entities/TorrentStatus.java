package com.bukanir.android.entities;

public class TorrentStatus {

    public String name;
    public String state;
    public String progress;
    public String download_rate;
    public String upload_rate;
    public String num_peers;
    public String num_seeds;
    public String total_seeds;
    public String total_peers;

    @Override
    public String toString() {
        StringBuilder result = new StringBuilder();
        String NL = System.getProperty("line.separator");

        result.append(((Object)this).getClass().getName() + " {" + NL);
        result.append("  name: " + this.name + NL);
        result.append("  state: " + this.state + NL);
        result.append("  progress: " + this.progress + NL);
        result.append("  download_rate: " + this.download_rate + NL);
        result.append("  upload_rate: " + this.upload_rate + NL);
        result.append("  num_peers: " + this.num_peers + NL);
        result.append("  num_seeds: " + this.num_seeds + NL);
        result.append("  total_seeds: " + this.total_seeds + NL);
        result.append("  total_peers: " + this.total_peers + NL);
        result.append("}" + NL);

        return result.toString();
    }
}
