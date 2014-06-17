package com.bukanir.android.scrapers;

import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.util.ArrayList;
import java.util.Arrays;

import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.jsoup.select.Elements;

public class Titlovi {
	
	private static final String URL_SEARCH = "http://en.titlovi.com/subtitles/subtitles.aspx?subtitle=%s";
	private static final String URL_DOWNLOAD = "http://titlovi.com/downloads/default.ashx?type=1&mediaid=%s";
	
	public ArrayList<ArrayList<String>> search(String query) throws Exception {

		if(query == null) {
			return null;
		}
		
		String encodedQuery = "";
		try {
			encodedQuery = URLEncoder.encode(query, "UTF-8");
		} catch (UnsupportedEncodingException e) {
			throw e;
		}
		
		final String url = String.format(URL_SEARCH, encodedQuery);
		return parseHTML(url);
	}
	
	private ArrayList<ArrayList<String>> parseHTML(String url) throws Exception {
		Document doc = Jsoup.connect(url).timeout(10*1000).get();
		Elements nodes = doc.select("li[class=listing]");
		
		if(nodes.isEmpty()) {
			return null;
		}
		
		ArrayList<ArrayList<String>> results = new ArrayList<ArrayList<String>>();
		
		Elements subtitles = nodes.select("div[class=title c1]");
		for(Element subtitle : subtitles) {
			String id, title, year, release, downloadLink;
			id = title = year = release = downloadLink = "";
			Element link = subtitle.select("a").first();
			Element spanYear = subtitle.select("span[class=year]").first();
			Element spanRelease = subtitle.select("span[class=release]").first();
			String href = link.attr("href");
			
			if(link != null) {
				title = link.text().trim();
				id = href.split("-")[href.split("-").length - 1].replaceAll("[^0-9]", "");
				downloadLink = String.format(URL_DOWNLOAD, id);
			}
			
			if(spanYear != null) {
				year = spanYear.text().replace("(", "").replace(")", "").trim();
			}
			
			if(spanRelease != null) {
				release = spanRelease.text().trim();
			}
			
			results.add(new ArrayList<String>(Arrays.asList(id, title, year, release, downloadLink)));
		}
		
		return results;
	}

}
