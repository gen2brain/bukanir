package bukanir

import (
	"log"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// tpbTop TPB top
func tpbTop(category int, host string) {
	defer func() {
		wgt.Done()
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

// tpbSearch TPB search
func tpbSearch(query string, page int, host string, media string) {
	defer func() {
		wgt.Done()
		if r := recover(); r != nil {
			log.Print("TPB: Recovered in tpbSearch")
		}
	}()

	var cats string
	if media == "all" {
		cats = "201,207,205,208"
	} else if media == "movies" {
		cats = "201,207"
	} else if media == "episodes" {
		cats = "205,208"
	}

	pb := NewTpb(host)
	results, err := pb.Search(query, page, cats)
	if err != nil {
		log.Printf("ERROR: TPB Search: %s\n", err.Error())
		return
	}

	torrents = append(torrents, results...)
}

// eztvSearch EZTV search
func eztvSearch(query string, host string) {
	defer func() {
		wgt.Done()
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

// tmdbSearchMovie TMDB search movie
func tmdbSearchMovie(t TTorrent, config *tmdbConfig) {
	defer func() {
		wg.Done()
		<-throttle

		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbSearchMovie")
		}
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

// tmdbSearchTv TMDB search tv show
func tmdbSearchTv(t TTorrent, config *tmdbConfig) {
	defer func() {
		wg.Done()
		<-throttle

		if r := recover(); r != nil {
			log.Print("TMDB: Recovered in tmdbSearch")
		}
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

// tmdbSummary TMDB summary
func tmdbSummary(id int, category int, season int, episode int) {
	defer func() {
		wg.Done()
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
	var directorId int = getDirectorId(res.Credits.Crew)
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
		getCastIds(casts),
		getGenre(genres),
		video,
		director,
		directorId,
		res.Vote_average,
		tagline,
		overview,
		res.Runtime,
		imdbId,
	}
}

// tmdbAutoComplete TMDB autocomplete
func tmdbAutoComplete(query string) {
	defer func() {
		wg.Done()
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

// tmdbPopular TMDB popular
func tmdbPopular() {
	defer func() {
		wg.Done()
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

// tmdbTopRated TMDB top rated
func tmdbTopRated() {
	defer func() {
		wg.Done()
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

// tmdbGetGenres TMDB genres
func tmdbGetGenres() {
	defer func() {
		wg.Done()
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

// tmdbByGenre TMDB movies by genre
func tmdbByGenre(id int, limit int, tpbHost string) {
	defer func() {
		wg.Done()
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

	var th = make(chan int, 3*runtime.NumCPU())

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

		sort.Sort(ByTSeeders(results))

		var t TTorrent
		for _, rt := range results {
			if r.Release_date != "" && rt.Year != "" {
				tmdbYear, _ := strconv.Atoi(getYear(r.Release_date))
				torrentYear, _ := strconv.Atoi(rt.Year)
				if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
					if getTitle(r.Title) == rt.FormattedTitle || getTitle(r.Original_title) == rt.FormattedTitle {
						t = rt
						break
					}
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

// tmdbWithCast TMDB movies by cast
func tmdbWithCast(id int, limit int, tpbHost string) {
	defer func() {
		wg.Done()
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
		mp, err := md.MoviesWithCast(id, n)
		if err != nil {
			log.Printf("ERROR: MoviesWithCast: %v\n", err.Error())
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

	var th = make(chan int, 3*runtime.NumCPU())

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

		sort.Sort(ByTSeeders(results))

		var t TTorrent
		for _, rt := range results {
			if r.Release_date != "" && rt.Year != "" {
				tmdbYear, _ := strconv.Atoi(getYear(r.Release_date))
				torrentYear, _ := strconv.Atoi(rt.Year)
				if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
					if getTitle(r.Title) == rt.FormattedTitle || getTitle(r.Original_title) == rt.FormattedTitle {
						t = rt
						break
					}
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
		bycast = append(bycast, movie)
	}

	for _, res := range m {
		th <- 1
		wgt.Add(1)
		go searchTorrents(res)
	}
	wgt.Wait()
}

// tmdbWithCrew TMDB movies by crew
func tmdbWithCrew(id int, limit int, tpbHost string) {
	defer func() {
		wg.Done()
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
		mp, err := md.MoviesWithCrew(id, n)
		if err != nil {
			log.Printf("ERROR: MoviesWithCast: %v\n", err.Error())
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

	var th = make(chan int, 3*runtime.NumCPU())

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

		sort.Sort(ByTSeeders(results))

		var t TTorrent
		for _, rt := range results {
			if r.Release_date != "" && rt.Year != "" {
				tmdbYear, _ := strconv.Atoi(getYear(r.Release_date))
				torrentYear, _ := strconv.Atoi(rt.Year)
				if tmdbYear == torrentYear || tmdbYear == torrentYear-1 || tmdbYear == torrentYear+1 {
					if getTitle(r.Title) == rt.FormattedTitle || getTitle(r.Original_title) == rt.FormattedTitle {
						t = rt
						break
					}
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
		bycrew = append(bycrew, movie)
	}

	for _, res := range m {
		th <- 1
		wgt.Add(1)
		go searchTorrents(res)
	}
	wgt.Wait()
}
