package main

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

// Browsing state, driven by the toolbar dropdowns.
var (
	media  = "all"
	sortBy = "seeders"
)

func currentYear() int { return time.Now().Year() }

// runSearch opens a grid tab with the results of a text query.
func runSearch(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	acGen.Add(1)
	hideSuggestions()
	openBrowse(query, func(ctx context.Context) ([]bukanir.TMovie, error) {
		return client.Search(ctx, query, bukanir.SearchOptions{Media: media, SortBy: sortBy})
	})
	iup.GetHandle("search").SetAttribute("VALUE", "")
}

func onSearchClick() { runSearch(iup.GetHandle("search").GetAttribute("VALUE")) }

func onTop(text string) {
	openBrowse("Top "+text, func(ctx context.Context) ([]bukanir.TMovie, error) {
		return client.Category(ctx, topCategory(text))
	})
}

// autocompleteLimit bounds how many suggestions are shown.
const autocompleteLimit = 10

// acGen drops autocomplete results that a newer keystroke has superseded.
var acGen atomic.Int64

// onSearchChanged fetches debounced autocomplete suggestions for the query.
func onSearchChanged(query string) {
	query = strings.TrimSpace(query)
	gen := acGen.Add(1)
	if len([]rune(query)) < 2 {
		hideSuggestions()
		return
	}
	go func() {
		time.Sleep(220 * time.Millisecond)
		if acGen.Load() != gen {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := client.Autocomplete(ctx, query)
		if err != nil || acGen.Load() != gen {
			return
		}
		if len(items) > autocompleteLimit {
			items = items[:autocompleteLimit]
		}
		iup.PostMessage(iup.GetHandle("suggest"), "", 0, items)
	}()
}

// Background-loaded lists cached behind the toolbar menus.
var (
	menuMu        sync.Mutex
	popularItems  []bukanir.TItem
	topRatedItems []bukanir.TItem
	genreList     []bukanir.TGenre
)

// loadMenus fetches the Popular/Top Rated/Genre data in the background.
func loadMenus() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if items, err := client.Popular(ctx); err == nil {
			menuMu.Lock()
			popularItems = items
			menuMu.Unlock()
		}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if items, err := client.TopRated(ctx); err == nil {
			menuMu.Lock()
			topRatedItems = items
			menuMu.Unlock()
		}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if genres, err := client.Genres(ctx); err == nil {
			menuMu.Lock()
			genreList = genres
			menuMu.Unlock()
		}
	}()
}

func cachedPopular() []bukanir.TItem {
	menuMu.Lock()
	defer menuMu.Unlock()
	return popularItems
}

func cachedTopRated() []bukanir.TItem {
	menuMu.Lock()
	defer menuMu.Unlock()
	return topRatedItems
}

func cachedGenres() []bukanir.TGenre {
	menuMu.Lock()
	defer menuMu.Unlock()
	return genreList
}

func topCategory(text string) bukanir.Category {
	switch text {
	case "Movies":
		return bukanir.CategoryMovies
	case "HD Movies":
		return bukanir.CategoryHDMovies
	case "TV Shows":
		return bukanir.CategoryTV
	case "HD TV Shows":
		return bukanir.CategoryHDTV
	}
	return bukanir.CategoryMovies
}

// reloaders holds each open tab's reload function, keyed by the tab holder.
var reloaders = map[iup.Ihandle]func(){}

func setReloader(tab iup.Ihandle, reload func()) { reloaders[tab] = reload }

// tabSort is a browse tab's last results and the order they are shown in.
type tabSort struct {
	movies []bukanir.TMovie
	sortBy string
}

// tabSorts holds the sort state of each open browse tab, keyed by the tab holder.
var tabSorts = map[iup.Ihandle]*tabSort{}

func setSortable(tab iup.Ihandle, by string) { tabSorts[tab] = &tabSort{sortBy: by} }

// showSorted shows a browse tab's results in its order; other grids show results as given.
func showSorted(tab iup.Ihandle, movies []bukanir.TMovie) {
	ts := tabSorts[tab]
	if ts == nil {
		showMovies(tab, movies)
		return
	}
	ts.movies = movies
	sorted := slices.Clone(movies)
	bukanir.SortMovies(sorted, ts.sortBy)
	showMovies(tab, sorted)
}

// resortCurrentTab applies sortBy to the selected browse tab.
func resortCurrentTab() {
	tab := currentTab()
	ts := tabSorts[tab]
	if ts == nil || ts.sortBy == sortBy {
		return
	}
	ts.sortBy = sortBy
	if ts.movies != nil {
		showSorted(tab, ts.movies)
	}
}

// syncSortButton makes the Sort button show the order of tab, the selected one.
func syncSortButton(tab iup.Ihandle) {
	if ts := tabSorts[tab]; ts != nil && ts.sortBy != sortBy {
		sortBy = ts.sortBy
		setButtonTitle("btnSort", sortLabel())
	}
}

// onRefreshClick reloads the content of the currently selected tab.
func onRefreshClick() {
	if r := reloaders[currentTab()]; r != nil {
		r()
	}
}

// loadTop returns the default landing grid.
func loadTop() ([]bukanir.TMovie, error) {
	return client.Category(context.Background(), bukanir.CategoryHDMovies)
}
