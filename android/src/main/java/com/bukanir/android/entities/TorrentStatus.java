package com.bukanir.android.entities;

public class TorrentStatus {

    public String name;
    public String state;
    public String state_str;
    public String error;
    public String progress;
    public String download_rate;
    public String upload_rate;
    public String total_download;
    public String total_upload;
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
        result.append("  state_str: " + this.state_str + NL);
        result.append("  error: " + this.error + NL);
        result.append("  progress: " + this.progress + NL);
        result.append("  download_rate: " + this.download_rate + NL);
        result.append("  upload_rate: " + this.upload_rate + NL);
        result.append("  total_download: " + this.total_download + NL);
        result.append("  total_upload: " + this.total_upload + NL);
        result.append("  num_peers: " + this.num_peers + NL);
        result.append("  num_seeds: " + this.num_seeds + NL);
        result.append("  total_seeds: " + this.total_seeds + NL);
        result.append("  total_peers: " + this.total_peers + NL);
        result.append("}" + NL);

        return result.toString();
    }
}
