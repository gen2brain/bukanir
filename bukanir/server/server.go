package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"bukanir"
)

const (
	appName    = "bukanir"
	appVersion = "1.9"
)

var (
	forceShutdown chan bool
	httpListener  net.Listener
)

func setServer(w http.ResponseWriter) {
	w.Header().Set("Server", fmt.Sprintf("%s/%s", appName, appVersion))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	http.Error(w, "403 Forbidden", http.StatusForbidden)
	return
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	js, err := json.Marshal(map[string]string{"status": "OK"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(js)
}

func handleCategory(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	category, _ := strconv.Atoi(r.FormValue("c"))
	limit, _ := strconv.Atoi(r.FormValue("l"))
	force, _ := strconv.Atoi(r.FormValue("f"))
	cacheDir := r.FormValue("t")
	cacheDays, _ := strconv.ParseInt(r.FormValue("d"), 10, 64)

	if bukanir.IsValidCategory(category) {
		js, err := bukanir.Category(category, limit, force, cacheDir, cacheDays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(js))
	} else {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	query := r.FormValue("q")
	limit, _ := strconv.Atoi(r.FormValue("l"))
	force, _ := strconv.Atoi(r.FormValue("f"))
	cacheDir := r.FormValue("t")
	cacheDays, _ := strconv.ParseInt(r.FormValue("d"), 10, 64)

	js, err := bukanir.Search(query, limit, force, cacheDir, cacheDays)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	id, _ := strconv.Atoi(r.FormValue("i"))
	category, _ := strconv.Atoi(r.FormValue("c"))
	season, _ := strconv.Atoi(r.FormValue("s"))
	episode, _ := strconv.Atoi(r.FormValue("e"))

	js, err := bukanir.Summary(id, category, season, episode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleSubtitle(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	movie := r.FormValue("m")
	year := r.FormValue("y")
	release := r.FormValue("r")
	language := r.FormValue("l")
	category, _ := strconv.Atoi(r.FormValue("c"))
	season, _ := strconv.Atoi(r.FormValue("s"))
	episode, _ := strconv.Atoi(r.FormValue("e"))
	imdbId := r.FormValue("i")

	js, err := bukanir.Subtitle(movie, year, release, language, category, season, episode, imdbId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleAutoComplete(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	query := r.FormValue("q")
	limit, _ := strconv.Atoi(r.FormValue("l"))

	js, err := bukanir.AutoComplete(query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(js))
}

func handleUnzipSubtitle(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	uri := r.FormValue("u")
	dest := r.FormValue("d")

	sub, err := bukanir.UnzipSubtitle(uri, dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(sub))
}

func handleTrailer(w http.ResponseWriter, r *http.Request) {
	setServer(w)

	videoId := r.FormValue("i")

	if videoId != "" {
		uri, err := bukanir.Trailer(videoId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if uri == "empty" {
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte(uri))
			return
		}
	} else {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}
}

func handleShutdown(w http.ResponseWriter, r *http.Request) {
	setServer(w)
	fmt.Fprintf(w, "OK")
	forceShutdown <- true
}

func startHTTP(bind string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/search", handleSearch)
	mux.HandleFunc("/category", handleCategory)
	mux.HandleFunc("/summary", handleSummary)
	mux.HandleFunc("/subtitle", handleSubtitle)
	mux.HandleFunc("/autocomplete", handleAutoComplete)
	mux.HandleFunc("/unzipsubtitle", handleUnzipSubtitle)
	mux.HandleFunc("/trailer", handleTrailer)
	mux.HandleFunc("/shutdown", handleShutdown)

	handler := http.Handler(mux)

	s := &http.Server{
		Addr:    bind,
		Handler: handler,
	}

	var err error
	httpListener, err = net.Listen("tcp4", bind)
	if err != nil {
		log.Printf("Error startHTTP: %v", err)
	} else {
		go s.Serve(httpListener)
	}
}

func loop() {
	forceShutdown = make(chan bool, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-forceShutdown:
			if httpListener != nil {
				httpListener.Close()
			}
			return
		case <-signalChan:
			forceShutdown <- true
		}
	}
}

func main() {
	b := flag.String("bind", ":7314", "Bind address")
	v := flag.Bool("verbose", false, "Show verbose output")
	flag.Parse()

	bukanir.SetVerbose(*v)
	startHTTP(*b)
	loop()
}
