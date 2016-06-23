package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	goresume "github.com/nii236/gopher-resume/models"
)

func errHandler(f func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}
	}
}

func findOne(doc *goquery.Selection, selector string) *goquery.Selection {
	var result *goquery.Selection
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		result = s
	})
	return result
}

func fetchThemes() ([]string, error) {
	doc, err := goquery.NewDocument("https://jsonresume.org/themes")
	if err != nil {
		return nil, err
	}

	themes := []string{}
	doc.Find("#themes .theme").Each(func(i int, s *goquery.Selection) {
		theme := s.Find(".col-sm-12.col-xs-6 a").First()
		link, ok := theme.Attr("href")
		if !ok {
			return
		}
		themes = append(themes, strings.TrimPrefix(link, "http://themes.jsonresume.org/theme/"))
	})

	return themes, err
}

func main() {
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/api/resume", errHandler(func(w http.ResponseWriter, r *http.Request) error {
		url := r.URL.Query().Get("url")
		w.Header().Add("Content-type", "application/json")
		if url == "" {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing URL"})
		}

		resume, err := resumeForURL(url)
		if err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}

		return json.NewEncoder(w).Encode(resume)
	})).Methods("GET")

	r.HandleFunc("/resume", errHandler(func(w http.ResponseWriter, r *http.Request) error {
		var body struct {
			URL   string
			Theme string
		}
		body.URL = r.FormValue("url")
		body.Theme = r.FormValue("theme")

		resume, err := resumeForURL(body.URL)
		if err != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}

		var postBody bytes.Buffer
		if err := json.NewEncoder(&postBody).Encode(map[string]goresume.Resume{"resume": resume}); err != nil {
			return err
		}

		log.Printf("posting resume %#v", resume)
		resp, err := http.Post(
			fmt.Sprintf("http://themes.jsonresume.org/theme/%s", body.Theme),
			"application/json",
			&postBody,
		)
		if err != nil {
			return err
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		return nil
	})).Methods("POST")

	r.HandleFunc("/", errHandler(func(w http.ResponseWriter, r *http.Request) error {
		themes, err := fetchThemes()
		if err != nil {
			return err
		}

		temp, err := template.ParseFiles("./index.html.tmpl")
		if err != nil {
			return err
		}

		return temp.Execute(w, struct {
			Themes []string
		}{Themes: themes})
	}))

	http.Handle("/", r)
	log.Printf("Listening on port %s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
