package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"cloud.google.com/go/translate"
	"github.com/gorilla/mux"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

type app struct {
	client           *translate.Client
	connectivityData map[int]map[string]language.Tag
	ctx              context.Context
}

func main() {
	var gAPIKey string
	flag.StringVar(&gAPIKey, "gApiKey", "", "google api key")
	flag.Parse()

	var err error
	var app app
	app.connectivityData = make(map[int]map[string]language.Tag)
	app.ctx = context.Background()
	app.client, err = translate.NewClient(app.ctx, option.WithAPIKey(gAPIKey))
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	r := mux.NewRouter()
	r.Handle("/transl8", app.transl8Handler()).Methods(http.MethodPost)

	log.Println("listening")
	log.Fatal(http.ListenAndServe(":9010", commonHeaders(r)))
}

type reqStruct struct {
	OrgID  string `json:"org_id"`
	Source struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Type  string `json:"type"`
	} `json:"source"`
	MessageID string `json:"id"`
	TS        int64  `json:"ts"`
	Raw       string `json:"raw"`
}

type respStruct struct {
	Raw string `json:"raw"`
}

func (a *app) echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := respStruct{
			Raw: fmt.Sprintf("PATH: %s  CONTENT: %s", r.RequestURI, string(body)),
		}

		respBody, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(respBody)
	}
}

func (a *app) transl8Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req reqStruct
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		responseString, err := a.routeRequest(req)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := respStruct{
			Raw: responseString,
		}

		respBody, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(respBody)
	}
}

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(w, r)
	})
}
