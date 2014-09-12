package com.bukanir.android.scrapers;

import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;

import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.jsoup.select.Elements;

import com.bukanir.android.utils.Utils;
import com.bukanir.android.entities.Release;

public class Podnapisi {
	
	private static final String URL_BASE = "http://www.podnapisi.net";
	private static final String URL_SEARCH = URL_BASE + "/en/ppodnapisi/search?sT=-1&sK=%s&sJ=%s&sY=%s&sAKA=0&sS=downloads&sO=desc";
	
	public ArrayList<ArrayList<String>> search(String query, String year, String release, String lang) throws Exception {

		if(query == null) {
			return null;
		}
		
		String encodedQuery = "";
		try {
			encodedQuery = URLEncoder.encode(query, "UTF-8");
		} catch (UnsupportedEncodingException e) {
			throw e;
		}
		
		final String url = String.format(URL_SEARCH, encodedQuery, lang, year);
		return parseHTML(url, release);
	}
	
	private ArrayList<ArrayList<String>> parseHTML(String url, String torrentRelease) throws Exception {
		Document doc = Jsoup.connect(url).timeout(10*1000).get();

		Elements nodes = doc.select("table[class=list first_column_title]");

		ArrayList<ArrayList<String>> results = new ArrayList<ArrayList<String>>();

		Elements trs = nodes.select("tr[class~= ]");

        if(trs.isEmpty()) {
            return null;
        }

		for(Element tr : trs) {
			String id, title, year, release, downloadLink;
			id = title = year = release = downloadLink = "";
			
			Element subtitle_page_link = tr.select("a[class=subtitle_page_link]").first();
			Element subtitle_year = subtitle_page_link.select("b").first();
			ArrayList<String> subtitle_releases = new ArrayList<String>(Arrays.asList(tr.select("span[class=release]").attr("html_title").split("<br/>")));
			
			ArrayList<Release> releases = new ArrayList<Release>();
			for(String rel : subtitle_releases) {
				String score = Utils.compareRelease(torrentRelease, rel);
				releases.add(new Release(rel, score));
			}
			Collections.sort(releases);
			release = releases.get(0).name;
			
			String downloadUrl = URL_BASE + subtitle_page_link.attr("href");
			Document downloadDoc = Jsoup.connect(downloadUrl).timeout(10*1000).get();
			Element downloadHref = downloadDoc.select("a[class=button big download]").first();
			downloadLink = URL_BASE + downloadHref.attr("href").replace("predownload", "download");
			
			id = downloadHref.attr("href").split("/")[downloadHref.attr("href").split("/").length - 1];
			title = subtitle_page_link.text().replace(subtitle_year.text(), "");
			year = subtitle_year.text().replace("(", "").replace(")", "");

			results.add(new ArrayList<String>(Arrays.asList(id, title, year, release, downloadLink)));
		}
		
		return results;
	}

}
