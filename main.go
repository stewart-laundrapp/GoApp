package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var indexTemplate = template.Must(template.ParseFiles("index.html"))

//JSON Response from the NEWS Api
type Source struct {
	ID   interface{} `json:"id"`
	Name string      `json:"name"`
}

type Article struct {
	Source      Source    `json:"source"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	URLToImage  string    `json:"urlToImage"`
	PublishedAt time.Time `json:"publishedAt"`
	Content     string    `json:"content"`
}

func (a *Article) FormatPublisherDate() string {
	year, month, day := a.PublishedAt.Date()
	return fmt.Sprintf("%d %v %d", day, month, year)
}

type Results struct {
	Status       string    `json:"status"`
	TotalResults int       `json:"totalResults"`
	Articles     []Article `json:"articles"`
}

/*
	- SearchKey is the query itself
	- NextPage allows us to navigate through the pages
	- TotalPages is the number of pages
	- Results is the current page
*/
type Search struct {
	SearchKey  string
	NextPage   int
	TotalPages int
	Results    Results
}

/*
Start of a method to reuse the code below to have 1 instead of 2
*/
//type URLObject struct {
//	Scheme     string
//	Opaque     string // encoded opaque data
//	Host       string // host or host:port
//	Path       string // path (relative paths may omit leading slash)
//	RawPath    string // encoded path hint (see EscapedPath method); added in Go 1.5
//	ForceQuery bool   // append a query ('?') even if RawQuery is empty; added in Go 1.7
//	RawQuery   string // encoded query values, without '?'
//	Fragment   string // fragment for references, without '#'
//}

//func (urlobject URLObject) getURLEndPoint(r *http.Request) int {
//	// Parse + String preserve the original encoding.
//	u, err := url.Parse(r.URL.String())
//	if err != nil {
//		log.Fatal(err)
//	}
//	url123 := &URLObject{}
//
//	url123.Path = u.Path
//	//endpoint := urlobject.Path
//	fmt.Println(url123.Path)
//
//	switch url123.Path {
//		case "/search":
//			return 1
//		case "/top-headlines":
//			return 2
//		default:
//			return http.StatusInternalServerError
//	}
//}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}
	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

/*
method to check if its the last page
- Currently the free api only allows 100 results which is why the total = 5
*/
func (s *Search) GoToEnd() int {
	var total = 5 - s.CurrentPage()
	return s.CurrentPage() + total
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexTemplate.Execute(w, nil)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	params := u.Query()
	searchKey := params.Get("q")
	page := params.Get("page")

	if page == "" {
		page = "1"
	}

	log.Println("Searching Query is: ", searchKey)
	log.Println("Result page is: ", page)

	search := &Search{}
	search.SearchKey = searchKey

	next, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "Unexpected server error", http.StatusInternalServerError)
		fmt.Println("next Atoi")
		return
	}

	search.NextPage = next
	//Can be between 0 and 100
	pageSize := 20

	endpoint := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=d519eae84522415684ef0fc68ee60564&sortBy=publishedAt&language=en", url.QueryEscape(search.SearchKey), pageSize, search.NextPage)
	resp, err := http.Get(endpoint)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&search.Results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	search.TotalPages = int(math.Ceil(float64(search.Results.TotalResults / pageSize)))
	if ok := !search.IsLastPage(); ok {
		search.NextPage++
	}

	err = indexTemplate.Execute(w, search)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

}

func topHeadlines(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}
	params := u.Query()
	page := params.Get("page")

	if page == "" {
		page = "1"
	}
	search := &Search{}
	var endpoint = fmt.Sprintf("https://newsapi.org/v2/top-headlines?sources=bbc-news&apiKey=d519eae84522415684ef0fc68ee60564")
	resp, err := http.Get(endpoint)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&search.Results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	search.TotalPages = int(math.Ceil(float64(search.Results.TotalResults)))

	err = indexTemplate.Execute(w, search)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

}

func main() {
	//!TODO NO LONGER BEING USED DUE TO DOCKER TESTING AND API KEY NEEDED WITHIN THE ENDPOINT
	//newsApiKey = flag.String("apikey", "", "Newsapi.org access key")
	//flag.Parse()

	//if *newsApiKey == "" {
	//	log.Fatal("api key must be set")
	//}
	// Create Server and Route Handlers

	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/top-headlines", topHeadlines)
	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8080", nil)
	//srv := &http.Server{
	//	Handler:      r,
	//	Addr:         ":8080",
	//	ReadTimeout:  10 * time.Second,
	//	WriteTimeout: 10 * time.Second,
	//}
	// Start Server
	//go func() {
	//	log.Println("Starting Server")
	//	if err := http.ListenAndServe(":8080", nil); err != nil {
	//		log.Fatal(err)
	//	}
	//}()

	// Graceful Shutdown
	//waitForShutdown(srv)
}

/*
Used from a docker-go server start
*/

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}
