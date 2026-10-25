//go:build media

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

var (
	favorites       []favorite // newest first
	favoritesView   iup.Ihandle
	favoriteActions = map[iup.Ihandle]func(){}
	favoriteButtons = map[iup.Ihandle]favoriteButton{}
)

// favorite is a saved movie or TV show; its torrents are looked up fresh when it is opened.
type favorite struct {
	ID           int              `json:"id"`
	TV           bool             `json:"tv"`
	Category     bukanir.Category `json:"category"`
	Title        string           `json:"title"`
	Year         int              `json:"year"`
	PosterSmall  string           `json:"posterSmall"`
	PosterMedium string           `json:"posterMedium"`
	PosterLarge  string           `json:"posterLarge"`
	PosterXLarge string           `json:"posterXLarge"`
	Added        time.Time        `json:"added"`
}

// favoriteButton is the star toggle on a details view, refreshed when the title is toggled elsewhere.
type favoriteButton struct {
	button iup.Ihandle
	movie  bukanir.TMovie
}

func favoritesPath() string {
	return filepath.Join(configDir, "favorites.json")
}

func loadFavorites() {
	data, err := os.ReadFile(favoritesPath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("ERROR: FAVORITES: %v", err)
		}
		return
	}
	if err := json.Unmarshal(data, &favorites); err != nil {
		log.Printf("ERROR: FAVORITES: %s: %v", favoritesPath(), err)
	}
}

func saveFavorites() {
	data, err := json.MarshalIndent(favorites, "", "  ")
	if err != nil {
		log.Printf("ERROR: FAVORITES: %v", err)
		return
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		log.Printf("ERROR: FAVORITES: %v", err)
		return
	}
	tmp := favoritesPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("ERROR: FAVORITES: %v", err)
		return
	}
	if err := os.Rename(tmp, favoritesPath()); err != nil {
		log.Printf("ERROR: FAVORITES: %v", err)
	}
}

// isShowEntry reports whether m stands for a whole TV show, as favorites do, rather than one episode.
func isShowEntry(m bukanir.TMovie) bool {
	return m.Category.IsTV() && m.Season == 0 && m.Episode == 0
}

func favoriteIndex(m bukanir.TMovie) int {
	tv := m.Category.IsTV()
	for i, f := range favorites {
		if f.ID == m.ID && f.TV == tv {
			return i
		}
	}
	return -1
}

func isFavorite(m bukanir.TMovie) bool {
	return favoriteIndex(m) >= 0
}

// toggleFavorite adds or removes the title of m; an episode stands for its show.
func toggleFavorite(m bukanir.TMovie) {
	if i := favoriteIndex(m); i >= 0 {
		favorites = slices.Delete(favorites, i, i+1)
		setStatus("Removed " + m.Title + " from favorites")
	} else {
		f := favorite{
			ID:           m.ID,
			TV:           m.Category.IsTV(),
			Category:     m.Category,
			Title:        m.Title,
			Year:         m.Year,
			PosterSmall:  m.PosterSmall,
			PosterMedium: m.PosterMedium,
			PosterLarge:  m.PosterLarge,
			PosterXLarge: m.PosterXLarge,
			Added:        time.Now(),
		}
		if f.TV {
			f.Category = bukanir.CategoryTV
		}
		favorites = slices.Insert(favorites, 0, f)
		setStatus("Added " + m.Title + " to favorites")
	}
	saveFavorites()

	on := isFavorite(m)
	for _, fb := range favoriteButtons {
		if fb.movie.ID == m.ID && fb.movie.Category.IsTV() == m.Category.IsTV() {
			setFavoriteIcon(fb.button, on)
		}
	}
	if reload, open := reloaders[favoritesView]; open && favoritesView != 0 {
		reload(context.Background())
	}
}

// setFavoriteIcon shows a filled star on a favorite's button and an outline otherwise.
func setFavoriteIcon(b iup.Ihandle, on bool) {
	icon := "star-outline"
	if on {
		icon = "star"
	}
	monoButtons[b] = icon
	b.SetAttribute("IMAGE", tintedName(icon))
}

// favoriteMovies lists the favorites as grid entries; shows carry no season or episode.
func favoriteMovies() []bukanir.TMovie {
	out := make([]bukanir.TMovie, len(favorites))
	for i, f := range favorites {
		out[i] = bukanir.TMovie{
			ID:           f.ID,
			Title:        f.Title,
			Year:         f.Year,
			PosterSmall:  f.PosterSmall,
			PosterMedium: f.PosterMedium,
			PosterLarge:  f.PosterLarge,
			PosterXLarge: f.PosterXLarge,
			Category:     f.Category,
		}
	}
	return out
}

// forgetFavoritesView drops the favorites state of a closing view.
func forgetFavoritesView(view iup.Ihandle) {
	if fb, ok := favoriteButtons[view]; ok {
		delete(monoButtons, fb.button)
	}
	delete(favoriteActions, view)
	delete(favoriteButtons, view)
	if view == favoritesView {
		favoritesView = 0
	}
}

// openFavorites shows the Favorites view, switching to it when it is already open.
func openFavorites() {
	if reload, open := reloaders[favoritesView]; open && favoritesView != 0 {
		focusView(favoritesView)
		reload(context.Background())
		return
	}
	holder := gridView()
	holder.SetAttribute("EMPTYTEXT", "No favorites yet. Use the star on a title to add one.")
	showList("Favorites", holder, true)
	favoritesView = holder
	reload := func(context.Context) { iup.PostMessage(holder, "", 0, favoriteMovies()) }
	setReloader(holder, reload)
	reload(context.Background())
}

// openShow lists the episodes of a favorite show, newest first.
func openShow(m bukanir.TMovie) {
	title, id := m.Title, m.ID
	openBrowseSorted(title, "episodes", false, func(ctx context.Context) ([]bukanir.TMovie, error) {
		found, err := client.Search(ctx, title, bukanir.SearchOptions{Media: "episodes"})
		if err != nil {
			return nil, err
		}
		return slices.DeleteFunc(found, func(e bukanir.TMovie) bool { return e.ID != id || !e.Category.IsTV() }), nil
	})
}

// findTorrent looks up the best torrent for a favorite movie, which is saved without one.
func findTorrent(ctx context.Context, m bukanir.TMovie) (bukanir.TMovie, bool) {
	query := m.Title
	if m.Year > 0 {
		query = fmt.Sprintf("%s %d", m.Title, m.Year)
	}
	movies, err := client.Search(ctx, query, bukanir.SearchOptions{Media: "movies"})
	if err != nil {
		log.Printf("FAVORITES: torrent lookup for %q: %v", m.Title, err)
		return m, false
	}
	for _, found := range movies {
		if found.ID == m.ID && !found.Category.IsTV() {
			return found, true
		}
	}
	return m, false
}

// favoriteMenu pops the Add/Remove favorite menu for a grid title at the pointer.
func favoriteMenu(m bukanir.TMovie) {
	label := "Add to Favorites"
	if isFavorite(m) {
		label = "Remove from Favorites"
	} else if m.Category.IsTV() && !isShowEntry(m) {
		label = "Add Show to Favorites"
	}
	menu := iup.Menu(menuItemAction(label, func() { toggleFavorite(m) }))
	iup.Popup(menu, iup.MOUSEPOS, iup.MOUSEPOS)
	iup.Destroy(menu)
}
