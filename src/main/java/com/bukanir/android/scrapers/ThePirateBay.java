package com.bukanir.android.scrapers;

import android.util.Log;

import com.bukanir.android.utils.Utils;

import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.text.ParseException;
import java.text.SimpleDateFormat;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Date;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.jsoup.select.Elements;

public class ThePirateBay {

    public String TPB_URL;
	public static final String TPB_HOST = "thepiratebay.se";
    public static final String TPB_HOST_PROXY1 = "thepiratebay.mg";
    public static final String TPB_HOST_PROXY2 = "thepiratebay.si";
    public static final String TPB_HOST_PROXY3 = "thepiratebay.je";
    public static final String TPB_HOST_PROXY4 = "pirateproxy.net";

	private static final String TOP_URL = "%s/top/%s";
	private static final String QUERY_URL = "%s/search/%s/0/%s/201,202,207";
	
	public static final String SORT_SEEDS = "7";
	
	public static final String CATEGORY_MOVIES = "201";
	public static final String CATEGORY_MOVIES_DVDR = "202";
	public static final String CATEGORY_HD_MOVIES = "207";

    public ThePirateBay(boolean proxy) {
        if(proxy) {
            if(Utils.isNetworkReachable(TPB_HOST)) {
                TPB_URL = "http://" + TPB_HOST;
            } else if(Utils.isNetworkReachable(TPB_HOST_PROXY1)) {
                TPB_URL = "http://" + TPB_HOST_PROXY1;
            } else if(Utils.isNetworkReachable(TPB_HOST_PROXY2)) {
                TPB_URL = "http://" + TPB_HOST_PROXY2;
            } else if(Utils.isNetworkReachable(TPB_HOST_PROXY3)) {
                TPB_URL = "http://" + TPB_HOST_PROXY3;
            } else {
                TPB_URL = "http://" + TPB_HOST_PROXY4;
            }
        } else {
            TPB_URL = "http://" + TPB_HOST;
        }
        Log.d("TPB_URL", TPB_URL);
    }
	
	public ArrayList<ArrayList<String>> search(String query, String order) throws Exception {

		if (query == null) {
			return null;
		}
		
		String encodedQuery = "";
		try {
			encodedQuery = URLEncoder.encode(query, "UTF-8");
		} catch (UnsupportedEncodingException e) {
			throw e;
		}
		
		final String url = String.format(QUERY_URL, TPB_URL, encodedQuery, order);
        Log.d("QUERY_URL", url);
		return parseHTML(url);
	}
	
	public ArrayList<ArrayList<String>> top(String category) throws Exception {
		final String url = String.format(TOP_URL, TPB_URL, category);
        Log.d("TOP_URL", url);
		return parseHTML(url);
	}
	
	private ArrayList<ArrayList<String>> parseHTML(String url) throws Exception {
		Document doc = Jsoup.connect(url).timeout(10*1000).get();
		Elements nodes = doc.select("div[class=detName]");
		
		ArrayList<ArrayList<String>> results = new ArrayList<ArrayList<String>>();
		
		for(Element node : nodes) {
			Element parent = node.parent();
			Element href = node.select("a[href]").first();
			Element font = parent.select("font[class=detDesc]").first();

			String name = href.text();
			String title = getTitle(name);
			String year = getYear(name);
			String details = String.format("%s%s", TPB_URL, href.attr("href"));
			String magnetLink = node.nextElementSibling().attr("href");
            String seeders = parent.nextElementSibling().text();
			String leechers = parent.nextElementSibling().nextElementSibling().text();
			
			String[] desc = font.text().split(",");
			String size = desc[1].replace(" Size ", "").replace("&nbsp;", " ");
			
			Date date = null;
			String dateText = desc[0].replace("Uploaded ", "").replace("&nbsp;", "");
			SimpleDateFormat dateformat = new SimpleDateFormat("MMddyyyy");
			
			try {
				date = dateformat.parse(dateText);
			} catch (ParseException e) {
			}
			
			String dateString = String.valueOf(date);
			
			results.add(new ArrayList<String>(Arrays.asList(title, year, name, magnetLink, details, size, dateString, seeders, leechers)));
		}
		
		return results;
		
	}

	private String getTitle(String torrentName) {
		String title = torrentName.replace(".", " ").replace("-", "").toLowerCase();
		Pattern pattern1 = Pattern.compile("(.*?)(dvdrip|xvid|dvdscr|brrip|bdrip|divx|klaxxon|hc|webrip|hdrip|hdtv|eztv|proper|720p|1080p|[\\{\\(\\[]?[0-9]{4}).*");
		Pattern pattern2 = Pattern.compile("(.*?)\\(.*\\)(.*)");
		
		Matcher matcher1 = pattern1.matcher(title);
		if(matcher1.find()) {
			title = matcher1.group(1);
		}

		Matcher matcher2 = pattern2.matcher(title);
		if (matcher2.find()) {
			title = matcher2.group(1);
		}

		return title.trim();
	}
	
	private String getYear(String torrentName) {
		String year = "";
		Pattern pattern = Pattern.compile("(.*)(19\\d{2}|20\\d{2})(.*)");
		Matcher matcher = pattern.matcher(torrentName);
		if(matcher.find()) {
			year = matcher.group(2);
		}
		return year;
	}

}
