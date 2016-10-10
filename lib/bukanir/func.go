package bukanir

import (
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Saves cache
func saveCache(key string, data []byte, cacheDir string) {
	md5key := md5.Sum([]byte(strings.ToLower(key)))
	file := filepath.Join(cacheDir, fmt.Sprintf("%x.json.gz", md5key))

	err := os.MkdirAll(cacheDir, 0777)
	if err != nil {
		log.Printf("ERROR: MkdirAll %s: %s\n", cacheDir, err.Error())
	}

	f, err := os.Create(file)
	if err != nil {
		log.Printf("ERROR: Create %s: %s\n", file, err.Error())
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	_, err = gz.Write(data)
	if err != nil {
		log.Printf("ERROR: Write: %s\n", err.Error())
	}
}

// Gets cache
func getCache(key string, cacheDir string, days int64) []byte {
	md5key := md5.Sum([]byte(strings.ToLower(key)))
	file := filepath.Join(cacheDir, fmt.Sprintf("%x.json.gz", md5key))

	info, err := os.Stat(file)
	if err != nil {
		return nil
	}

	if days != 0 && days > 0 {
		mtime := info.ModTime().Unix()
		if time.Now().Unix()-mtime > 86400*days {
			return nil
		}
	}

	if verbose {
		log.Printf("BUK: Using cache file %s\n", file)
	}

	f, err := os.Open(file)
	if err != nil {
		log.Printf("ERROR: Open %s: %s\n", file, err.Error())
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Printf("ERROR: NewReader %s: %s\n", file, err.Error())
	}
	defer gz.Close()

	data, err := ioutil.ReadAll(gz)
	if err != nil {
		log.Printf("ERROR: ReadAll: %v\n", err.Error())
		return nil
	}
	return data
}

// TPB top
func tpbTop(category int, host string) {
	defer wgt.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TPB: Recovered in tpbTop")
		}
	}()

	pb := NewTpb(host)
	results, err := pb.Top(category)
	if err != nil {
		log.Printf("ERROR: TPB Top: %v\n", err.Error())
		return
	}

	torrents = append(torrents, results...)
}

// TPB search
func tpbSearch(query string, page int, host string) {
	defer wgt.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TPB: Recovered in tpbSearch")
		}
	}()

	pb := NewTpb(host)
	results, err := pb.Search(query, page, "201,207,205,208")
	if err != nil {
		log.Printf("ERROR: TPB Search: %s\n", err.Error())
		return
	}

	torrents = append(torrents, results...)
}

// EZTV search
func eztvSearch(query string, host string) {
	defer wgt.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TPB: Recovered in eztvSearch")
		}
	}()

	ez := NewEztv(host)
	results, err := ez.Search(query)
	if err != nil {
		log.Printf("ERROR: EZTV Search: %s\n", err.Error())
		return
	}

	torrents = append(torrents, results...)
}

// TMDB search movie
func tmdbSearchMovie(t tTorrent, config *tmdbConfig) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbSearchMovie")
		}
	}()

	defer func() {
		<-throttle
	}()

	md := NewTmdb(tmdbApiKey)

	results, err := md.SearchMovie(t.FormattedTitle)
	if err != nil {
		log.Printf("ERROR: TMDB Search: %s\n", err.Error())
		return
	}

	if results.Total_results == 0 {
		return
	}

	res := new(tmdbResult)
	for _, result := range results.Results {
		if result.Release_date != "" && t.Year != "" {
			tmdbYear, _ := strconv.Atoi(getYear(result.Release_date))
			torrentYear, _ := strconv.Atoi(t.Year)
			if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
				res = &result
				break
			}
		}
	}

	if res.Id == 0 || len(config.Images.Poster_sizes) < 5 {
		return
	}

	posterSmall := config.Images.Base_url + config.Images.Poster_sizes[0] + res.Poster_path
	posterMedium := config.Images.Base_url + config.Images.Poster_sizes[1] + res.Poster_path
	posterLarge := config.Images.Base_url + config.Images.Poster_sizes[3] + res.Poster_path
	posterXLarge := config.Images.Base_url + config.Images.Poster_sizes[4] + res.Poster_path

	m := TMovie{
		res.Id,
		res.Title,
		getYear(res.Release_date),
		posterSmall,
		posterMedium,
		posterLarge,
		posterXLarge,
		int64(t.Size),
		t.SizeHuman,
		t.Seeders,
		t.MagnetLink,
		t.Title,
		t.Category,
		t.Season,
		t.Episode,
		getQuality(t.Title),
	}
	movies = append(movies, m)
}

// TMDB search tv show
func tmdbSearchTv(t tTorrent, config *tmdbConfig) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbSearch")
		}
	}()

	defer func() {
		<-throttle
	}()

	md := NewTmdb(tmdbApiKey)

	results, err := md.SearchTv(t.FormattedTitle)
	if err != nil {
		log.Printf("ERROR: TMDB Search: %s\n", err.Error())
		return
	}

	if results.Total_results == 0 {
		return
	}

	res := &results.Results[0]
	if res.Id == 0 {
		return
	}

	p, err := md.GetTvImages(strconv.Itoa(res.Id), t.Season)
	if err != nil {
		log.Printf("ERROR: GetTvImages: %s\n", err.Error())
		return
	}

	if len(config.Images.Poster_sizes) < 5 || len(p.Posters) < 1 {
		return
	}

	posterSmall := config.Images.Base_url + config.Images.Poster_sizes[0] + p.Posters[0].File_path
	posterMedium := config.Images.Base_url + config.Images.Poster_sizes[1] + p.Posters[0].File_path
	posterLarge := config.Images.Base_url + config.Images.Poster_sizes[3] + p.Posters[0].File_path
	posterXLarge := config.Images.Base_url + config.Images.Poster_sizes[4] + p.Posters[0].File_path

	m := TMovie{
		res.Id,
		res.Name,
		getYear(res.First_air_date),
		posterSmall,
		posterMedium,
		posterLarge,
		posterXLarge,
		int64(t.Size),
		t.SizeHuman,
		t.Seeders,
		t.MagnetLink,
		t.Title,
		t.Category,
		t.Season,
		t.Episode,
		getQuality(t.Title),
	}
	movies = append(movies, m)
}

// TMDB summary
func tmdbSummary(id int, category int, season int, episode int) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbSummary")
		}
	}()

	md := NewTmdb(tmdbApiKey)

	var err error
	var res, res_season tmdbMovie

	if category == CategoryTV || category == CategoryHDTV {
		res, err = md.GetTvDetails(strconv.Itoa(id), -1)
		if err != nil {
			log.Printf("ERROR: GetTvDetails: %v\n", err.Error())
			return
		}
		res_season, err = md.GetTvDetails(strconv.Itoa(id), season)
		if err != nil {
			log.Printf("ERROR: GetTvDetails season: %v\n", err.Error())
			return
		}
	} else {
		res, err = md.GetMovieDetails(strconv.Itoa(id))
		if err != nil {
			log.Printf("ERROR: GetMovieDetails: %v\n", err.Error())
			return
		}
	}

	var tagline string = res.Tagline
	var overview string = res.Overview
	var director string = getDirector(res.Credits.Crew)
	var video = getVideo(res.Videos.Results)
	var casts []tmdbCast = res.Credits.Cast
	var genres []tmdbGenre = res.Genres
	var imdbId string = strings.Replace(res.Imdb_id, "tt", "", -1)

	if category == CategoryTV || category == CategoryHDTV {
		if len(res_season.Episodes) > episode-1 {
			o := res_season.Episodes[episode-1].Overview
			if o != "" {
				overview = o
			}

			if len(res_season.Episodes[episode-1].Crew) > 0 {
				d := getDirector(res_season.Episodes[episode-1].Crew)
				if d != "" {
					director = d
				}
			}
		}

		if len(res_season.Videos.Results) > 0 {
			v := getVideo(res_season.Videos.Results)
			if v != "" {
				video = v
			}
		}

		if len(res_season.Credits.Cast) > 0 {
			casts = res_season.Credits.Cast
		}

		if res_season.Tagline != "" {
			tagline = res_season.Tagline
		}

		if imdbId == "" {
			ext, err := md.GetTvExternals(strconv.Itoa(id))
			if err != nil {
				log.Printf("ERROR: GetTvExternals: %v\n", err.Error())
				return
			}
			imdbId = strings.Replace(ext.Imdb_id, "tt", "", -1)
		}
	}

	details = TSummary{
		id,
		getCast(casts),
		getGenre(genres),
		video,
		director,
		res.Vote_average,
		tagline,
		overview,
		res.Runtime,
		imdbId,
	}
}

// TMDB autocomplete
func tmdbAutoComplete(query string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbAutocomplete")
		}
	}()

	md := NewTmdb(tmdbApiKey)

	movies, err := md.AutoCompleteMovie(query)
	if err != nil {
		log.Printf("ERROR: AutoCompleteMovie: %v\n", err.Error())
		return
	}
	tvs, err := md.AutoCompleteTv(query)
	if err != nil {
		log.Printf("ERROR: AutoCompleteTv: %v\n", err.Error())
		return
	}

	if tvs.Total_results+movies.Total_results == 0 {
		return
	}

	for _, movie := range movies.Results {
		year := getYear(movie.Release_date)
		if year == "" {
			continue
		}
		a := TItem{
			movie.Title,
			year,
		}
		autocompletes = append(autocompletes, a)
	}

	for _, tv := range tvs.Results {
		year := getYear(tv.First_air_date)
		if year == "" {
			continue
		}
		a := TItem{
			tv.Original_name,
			year,
		}
		autocompletes = append(autocompletes, a)
	}
}

// TMDB popular
func tmdbPopular() {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbPopular")
		}
	}()

	md := NewTmdb(tmdbApiKey)

	movies, err := md.PopularMovies()
	if err != nil {
		log.Printf("ERROR: PopularMovies: %v\n", err.Error())
		return
	}

	tvs, err := md.PopularTv()
	if err != nil {
		log.Printf("ERROR: PopularTv: %v\n", err.Error())
		return
	}

	if movies.Total_results+tvs.Total_results == 0 {
		return
	}

	for _, movie := range movies.Results {
		p := TItem{}
		p.Title = movie.Title
		p.Year = getYear(movie.Release_date)
		popular = append(popular, p)
	}

	sep := TItem{}
	popular = append(popular, sep)

	for _, tv := range tvs.Results {
		t := TItem{}
		t.Title = tv.Name
		t.Year = getYear(tv.First_air_date)
		popular = append(popular, t)
	}
}

// TMDB top rated
func tmdbTopRated() {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbTopRated")
		}
	}()

	md := NewTmdb(tmdbApiKey)

	movies, err := md.TopRatedMovies()
	if err != nil {
		log.Printf("ERROR: TopRatedMovies: %v\n", err.Error())
		return
	}

	tvs, err := md.TopRatedTv()
	if err != nil {
		log.Printf("ERROR: TopRatedTv: %v\n", err.Error())
		return
	}

	if movies.Total_results+tvs.Total_results == 0 {
		return
	}

	for _, movie := range movies.Results {
		p := TItem{}
		p.Title = movie.Title
		p.Year = getYear(movie.Release_date)
		toprated = append(toprated, p)
	}

	sep := TItem{}
	toprated = append(toprated, sep)

	for _, tv := range tvs.Results {
		t := TItem{}
		t.Title = tv.Name
		t.Year = getYear(tv.First_air_date)
		toprated = append(toprated, t)
	}
}

// TMDB genres
func tmdbGetGenres() {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbGenres")
		}
	}()

	md := NewTmdb(tmdbApiKey)

	gn, err := md.Genres()
	if err != nil {
		log.Printf("ERROR: Genres: %v\n", err.Error())
		return
	}

	if len(gn.Genres) == 0 {
		return
	}

	for _, g := range gn.Genres {
		p := TGenre{g.Id, g.Name}
		genres = append(genres, p)
	}
}

// TMDB movies by genre
func tmdbByGenre(id int, limit int, tpbHost string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbByGenre")
		}
	}()

	md := NewTmdb(tmdbApiKey)
	config, err := md.GetConfig()
	if err != nil {
		log.Printf("ERROR: TMDB GetConfig: %s\n", err.Error())
		return
	}

	var m []tmdbResult
	for n := 1; n < 6; n++ {
		mp, err := md.Genre(id, n)
		if err != nil {
			log.Printf("ERROR: Genre: %v\n", err.Error())
			return
		}
		m = append(m, mp.Results...)
	}

	if len(m) == 0 {
		return
	}

	pb := NewTpb(tpbHost)

	if limit > 0 {
		if limit > len(m) {
			limit = len(m)
		}
		m = m[0:limit]
	}

	var th = make(chan int, 15)

	searchTorrents := func(r tmdbResult) {
		defer wgt.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Print("TMDB: Recovered in searchTorrents")
			}
		}()

		defer func() {
			<-th
		}()

		if r.Id == 0 {
			return
		}

		results, err := pb.Search(r.Title, 0, "201,207")
		if err != nil {
			return
		}

		if len(results) == 0 {
			return
		}

		sort.Sort(byTSeeders(results))

		var t tTorrent
		for _, rt := range results {
			if r.Release_date != "" && rt.Year != "" {
				tmdbYear, _ := strconv.Atoi(getYear(r.Release_date))
				torrentYear, _ := strconv.Atoi(rt.Year)
				if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
					t = rt
					break
				}
			}
		}

		if t.Title == "" {
			return
		}

		movie := TMovie{
			r.Id,
			r.Title,
			getYear(r.Release_date),
			config.Images.Base_url + config.Images.Poster_sizes[0] + r.Poster_path,
			config.Images.Base_url + config.Images.Poster_sizes[1] + r.Poster_path,
			config.Images.Base_url + config.Images.Poster_sizes[3] + r.Poster_path,
			config.Images.Base_url + config.Images.Poster_sizes[4] + r.Poster_path,
			int64(t.Size),
			t.SizeHuman,
			t.Seeders,
			t.MagnetLink,
			t.Title,
			t.Category,
			t.Season,
			t.Episode,
			getQuality(t.Title),
		}
		bygenre = append(bygenre, movie)
	}

	for _, res := range m {
		th <- 1
		wgt.Add(1)
		go searchTorrents(res)
	}
	wgt.Wait()
}
