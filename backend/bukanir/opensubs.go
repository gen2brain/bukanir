package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"reflect"
	"sort"
	"strconv"

	xmlrpc "github.com/sqp/go-xmlrpc"
)

const OPENSUBTITLE_DOMAIN = "http://api.opensubtitles.org/xml-rpc"

type SubInfo struct {
	MatchedBy        string
	MovieHash        string
	MovieName        string
	MovieYear        string
	MovieReleaseName string
	IDSubtitleFile   string
	SubLanguageID    string
	SubFormat        string
	SubFileName      string
	SubAddDate       string
	SubDownloadsCnt  string
	SeriesSeason     string
	SeriesEpisode    string
	IDMovieImdb      string
	UserNickName     string
	UserRank         string
	ZipDownloadLink  string
	reader           io.Reader
}

func (sub SubInfo) Id() int {
	i, _ := strconv.Atoi(sub.IDSubtitleFile)
	return i
}

func (sub SubInfo) ByHash() bool {
	return sub.MatchedBy == "moviehash"
}

func (sub SubInfo) Reader() io.Reader {
	return sub.reader
}

// Map SubInfo by their subId. Used to match downloaded subs.
type subIndex map[string]*SubInfo

// The list of available subs matched for one language of current reference.
// Can be sorted. Current sort method is byDownloads. More could be added easily.
type subsList []*SubInfo

func (s subsList) Len() int      { return len(s) }
func (s subsList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Sort method for subsList: Order subs by downloaded count. Highest is first.
type byDownloads struct{ subsList }

func (s byDownloads) Less(i, j int) bool {
	vi, _ := strconv.Atoi(s.subsList[i].SubDownloadsCnt)
	vj, _ := strconv.Atoi(s.subsList[j].SubDownloadsCnt)
	return vi > vj
}

// First level of maping.
type subByLang map[string]subsList

// Second level of maping.
type subByRef map[string]subByLang

func (byref subByRef) addSub(sub *SubInfo, key string) {
	if _, ok := byref[key]; !ok {
		byref[key] = make(subByLang)
	}
	byref[key][sub.SubLanguageID] = append(byref[key][sub.SubLanguageID], sub)
}

//-----------------------------------------------------------------------
// Query public API.
//-----------------------------------------------------------------------

type Query struct {
	listArgs  []interface{}
	byhash    subByRef
	byimdb    subByRef
	hashs     map[string]string // Index to rematch subs with files.
	userAgent string
	token     string
}

func OpenSubsNewQuery(userAgent string) *Query {
	return &Query{
		hashs:     make(map[string]string),
		userAgent: userAgent,
	}
}

func (q *Query) AddImdb(imdb, langs string) *Query {
	q.listArgs = append(q.listArgs, map[string]string{"sublanguageid": langs, "imdbid": imdb})
	return q
}

func (q *Query) AddQuery(query string) *Query {
	q.listArgs = append(q.listArgs, map[string]string{"query": query})
	return q
}

func (q *Query) AddSeason(season string) *Query {
	q.listArgs = append(q.listArgs, map[string]string{"season": season})
	return q
}

func (q *Query) AddEpisode(episode string) *Query {
	q.listArgs = append(q.listArgs, map[string]string{"episode": episode})
	return q
}

func (q *Query) Search() error {
	return q.search()
}

func (q *Query) Get(n int) (subByRef, subByRef) {
	var dl []string
	needed := make(subIndex)

	// Parsing list byimdb to get multiple files.
	for _, bylang := range q.byimdb { // For each movie
		for _, list := range bylang { // For each lang

			sort.Sort(byDownloads{list})
			count := 0

			for _, sub := range list { // each sub
				if n == -1 || count < n { // Unlimited or within limit: add to list.
					needed[sub.IDSubtitleFile] = sub
					dl = append(dl, sub.IDSubtitleFile)
				}
				count++
			}
		}
	}
	return q.download(dl, needed)
}

// Close the token on the server.
func (q *Query) Logout() {
	call("LogOut", q.token)
}

//-----------------------------------------------------------------------
// Server query.
//-----------------------------------------------------------------------

// Process a xmlrpc call on OpenSubtitles.org server.
func call(name string, args ...interface{}) (xmlrpc.Struct, error) {
	res, e := xmlrpc.Call(OPENSUBTITLE_DOMAIN, name, args...)
	if e == nil {
		if data, ok := res.(xmlrpc.Struct); ok {
			return data, e
		}
	}
	return nil, e
}

// Initiate connection to OpenSubtitles.org to get a valid token.
func (q *Query) connect() error {
	res, e := call("LogIn", "", "", "en", q.userAgent)
	switch {
	case e != nil:
		return e
	case res == nil || len(res) == 0:
		return errors.New("connection problem")
	}

	if token, ok := res["token"].(string); ok {
		q.token = token
		return nil
	}
	return errors.New("OpenSubtitles Token problem")
}

func (q *Query) search() error {
	e := q.connect()
	switch {
	case e != nil:
		return e
	case q.token == "":
		return errors.New("invalid token")
	}

	searchData, e := call("SearchSubtitles", q.token, q.listArgs)
	if e != nil {
		return e
	}
	for k, v := range searchData {
		if k == "data" {
			if array, ok := v.(xmlrpc.Array); ok {
				q.byhash, q.byimdb = mapSubInfos(array)
			}
		}
	}

	return nil
}

func (q *Query) download(ids []string, needed subIndex) (subByRef, subByRef) {
	if len(ids) == 0 {
		return nil, nil
	}
	if s, e := call("DownloadSubtitles", q.token, ids); e == nil {
		for k, v := range s {
			if k == "data" {
				if array, ok := v.(xmlrpc.Array); ok { // Found valid data array.
					return q.parseSubFiles(array, needed)
				}
			}
		}
	}
	return nil, nil
}

//-----------------------------------------------------------------------
// Parse downloaded files.
//-----------------------------------------------------------------------

func (q *Query) parseSubFiles(array xmlrpc.Array, needed subIndex) (subByRef, subByRef) {
	byhash := make(subByRef)
	byimdb := make(subByRef)

	var subid, subtext string
	var gz []byte
	var e error
	var reader io.Reader
	var sub *SubInfo

	for _, fi := range array {
		data, ok := fi.(xmlrpc.Struct)
		if !ok {
			continue
		}
		subid, ok = data["idsubtitlefile"].(string)
		if !ok {
			continue
		}

		subtext, ok = data["data"].(string)
		if !ok {
			continue
		}

		/// Get matching SubInfo
		sub, ok = needed[subid]
		if !ok {
			continue
		}

		/// unbase64
		gz, e = base64.StdEncoding.DecodeString(subtext)
		if e != nil || len(gz) == 0 {
			warn("base64", e)
			continue
		}
		reader = bytes.NewBuffer(gz)

		/// gunzip
		reader, e = gzip.NewReader(reader)
		if e != nil {
			warn("gunzip", e)
			continue
		}

		sub.reader = reader
		if sub.SubFormat != "srt" {
			warn("sub format", sub.SubFormat)
		}

		/// Everything was OK: add the reference to result.
		switch sub.MatchedBy {
		case "moviehash":
			byhash.addSub(sub, q.hashs[sub.MovieHash])
		case "imdbid":
			byimdb.addSub(sub, sub.IDMovieImdb)
		}
	}

	return byhash, byimdb
}

//-----------------------------------------------------------------------
// Parse downloaded SubInfo.
//-----------------------------------------------------------------------

func mapSubInfos(data []interface{}) (subByRef, subByRef) {
	byhash := make(subByRef)
	byimdb := make(subByRef)

	hashImdbIndex := make(subIndex)
	var matchedImdb subsList
	for _, value := range data { // Array of data
		if vMap, ok := value.(xmlrpc.Struct); ok {

			sub := mapOneSub(vMap)
			switch sub.MatchedBy {
			case "moviehash":
				byhash.addSub(sub, sub.MovieHash)
				hashImdbIndex[sub.IDMovieImdb] = sub // saving reference for 2nd pass
			case "imdbid":
				matchedImdb = append(matchedImdb, sub)
			default:
				warn("match failed. not implemented", sub.MatchedBy)
			}
		}
	}

	for _, sub := range matchedImdb {
		if _, ok := hashImdbIndex[sub.IDMovieImdb]; !ok { // Add to imdb list only if they were not already matched by hash.
			byimdb.addSub(sub, sub.IDMovieImdb)
		}
	}
	return byhash, byimdb
}

func mapOneSub(parseMap map[string]interface{}) *SubInfo {
	typ := reflect.TypeOf(SubInfo{})
	n := typ.NumField()

	item := &SubInfo{}
	elem := reflect.ValueOf(item).Elem()

	for i := 0; i < n-1; i++ { // Parsing all fields in type except last one. reader is a private member.
		field := typ.Field(i)
		if v, ok := parseMap[field.Name]; ok { // Got matching row in map
			if elem.Field(i).Kind() == reflect.TypeOf(v).Kind() { // Types are compatible.
				elem.Field(i).Set(reflect.ValueOf(v))
			} else {
				warn("XML Import Field mismatch", field.Name, elem.Field(i).Kind(), reflect.TypeOf(v).Kind())
			}
		}
	}
	return item
}

func warn(source string, data ...interface{}) {
	args := []interface{}{}
	args = append(args, source)
	args = append(args, data...)
	log.Println(args...)
}
