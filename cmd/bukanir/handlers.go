//go:build media

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

// autocompleteLimit bounds how many suggestions are shown.
const autocompleteLimit = 10

// Browsing state, driven by the toolbar dropdowns.
var (
	media  = "all"
	sortBy = "seeders"
)

// acGen drops autocomplete results that a newer keystroke has superseded.
var acGen atomic.Int64

// Background-loaded lists cached behind the toolbar menus.
var (
	menuMu        sync.Mutex
	popularItems  []bukanir.TItem
	topRatedItems []bukanir.TItem
	genreList     []bukanir.TGenre
)

// reloaders holds each open view's reload function, keyed by the view holder.
var reloaders = map[iup.Ihandle]func(context.Context){}

// viewSorts holds the sort state of each open browse view, keyed by the view holder.
var viewSorts = map[iup.Ihandle]*viewSort{}

func currentYear() int { return time.Now().Year() }

// runSearch opens a grid view with the results of a text query.
func runSearch(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	acGen.Add(1)
	endSearch()
	openRootBrowse(query, func(ctx context.Context) ([]bukanir.TMovie, error) {
		return client.Search(ctx, query, bukanir.SearchOptions{Media: media, SortBy: sortBy})
	})
}

func onTop(text string) {
	openRootBrowse("Top "+text, func(ctx context.Context) ([]bukanir.TMovie, error) {
		return client.Category(ctx, topCategory(text))
	})
}

// onSearchChanged fetches debounced autocomplete suggestions for the query.
func onSearchChanged(query string) {
	query = strings.TrimSpace(query)
	gen := acGen.Add(1)
	if len([]rune(query)) < 2 {
		hideSuggestions()
		return
	}
	media := media
	go func() {
		time.Sleep(220 * time.Millisecond)
		if acGen.Load() != gen {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		items, err := client.Autocomplete(ctx, query, media)
		if err != nil || acGen.Load() != gen {
			return
		}
		if len(items) > autocompleteLimit {
			items = items[:autocompleteLimit]
		}
		iup.PostMessage(bus(), "suggest", 0, items)
	}()
}

// loadMenus fetches the Popular/Top Rated/Genre data in the background.
func loadMenus() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if items, err := client.Popular(ctx); err == nil {
			menuMu.Lock()
			popularItems = items
			menuMu.Unlock()
			runOnMain(menusLoaded)
		}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if items, err := client.TopRated(ctx); err == nil {
			menuMu.Lock()
			topRatedItems = items
			menuMu.Unlock()
			runOnMain(menusLoaded)
		}
	}()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if genres, err := client.Genres(ctx); err == nil {
			menuMu.Lock()
			genreList = genres
			menuMu.Unlock()
			runOnMain(menusLoaded)
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

func setReloader(view iup.Ihandle, reload func(context.Context)) { reloaders[view] = reload }

// viewSort is a browse view's last results and the order they are shown in.
type viewSort struct {
	movies []bukanir.TMovie
	sortBy string
}

func setSortable(view iup.Ihandle, by string) { viewSorts[view] = &viewSort{sortBy: by} }

// showSorted shows a browse view's results in its order; other grids show results as given.
func showSorted(view iup.Ihandle, movies []bukanir.TMovie) {
	ts := viewSorts[view]
	if ts == nil {
		showMovies(view, movies)
		return
	}
	ts.movies = movies
	sorted := slices.Clone(movies)
	bukanir.SortMovies(sorted, ts.sortBy)
	showMovies(view, sorted)
}

// resortCurrentView applies sortBy to the selected browse view.
func resortCurrentView() {
	view := currentView()
	ts := viewSorts[view]
	if ts == nil || ts.sortBy == sortBy {
		return
	}
	ts.sortBy = sortBy
	if ts.movies != nil {
		showSorted(view, ts.movies)
	}
}

// syncSortButton makes the Sort button show the order of view, the selected one.
func syncSortButton(view iup.Ihandle) {
	if ts := viewSorts[view]; ts != nil && ts.sortBy != sortBy {
		sortBy = ts.sortBy
		showFilters()
	}
}

// onRefreshClick reloads the content of the currently selected view.
func onRefreshClick() {
	if r := reloaders[currentView()]; r != nil {
		r(bukanir.Refresh(context.Background()))
	}
}

// loadTop returns the default landing grid.
func loadTop(ctx context.Context) ([]bukanir.TMovie, error) {
	return client.Category(ctx, bukanir.CategoryHDMovies)
}
