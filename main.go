package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"cloud.google.com/go/translate"
	"github.com/gorilla/mux"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

type app struct {
	client *translate.Client
}

type room struct {
	ID   int
	lang language.Tag
}

var roomLanguage map[string][]room

func main() {
	var gAPIKey string
	flag.StringVar(&gAPIKey, "gApiKey", "", "google api key")
	flag.Parse()

	var err error
	var app app
	app.client, err = translate.NewClient(context.Background(), option.WithAPIKey(gAPIKey))
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.Handle("/translate", app.translateHandler()).Methods(http.MethodPost)

	log.Println("listening")
	log.Fatal(http.ListenAndServe(":9010", commonHeaders(r)))
}

func create(r room) (string, error) {
	return "", nil
}

func join(conferenceID string, r room) error {
	return nil
}

func leave(conferenceID string, r room) error {
	return nil
}

func (a *app) translateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type reqStruct struct {
			OrgID  string `json:"org_id"`
			Source struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
				Type  string `json:"type"`
			} `json:"source"`
			MessageID string `json:"id"`
			TS        string `json:"ts"`
			Raw       string `json:"raw"`
		}
		var req reqStruct

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		type respStruct struct {
			Raw string `json:"raw"`
		}

		respText, err := a.client.Translate(context.Background(), []string{req.Raw}, language.Polish, nil)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := respStruct{
			Raw: respText[0].Text,
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
