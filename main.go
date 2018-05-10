package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
}

func main() {
	var gAPIKey string
	flag.StringVar(&gAPIKey, "gApiKey", "", "google api key")
	flag.Parse()

	var err error
	var app app
	app.connectivityData = make(map[int]map[string]language.Tag)
	app.client, err = translate.NewClient(context.Background(), option.WithAPIKey(gAPIKey))
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	r := mux.NewRouter()
	r.Handle("/translate", app.translateHandler()).Methods(http.MethodPost)

	log.Println("listening")
	log.Fatal(http.ListenAndServe(":9010", commonHeaders(r)))
}

func (a *app) create(roomID string, lang language.Tag) int {
	ID := rand.Intn(100000)
	_, ok := a.connectivityData[ID]
	if ok {
		return a.create(roomID, lang)
	}
	a.connectivityData[ID] = make(map[string]language.Tag)
	a.connectivityData[ID][roomID] = lang

	return ID
}

func (a *app) join(conferenceID int, roomID string, lang language.Tag) error {
	_, ok := a.connectivityData[conferenceID]
	if !ok {
		return fmt.Errorf("conference %d doesn't exist", conferenceID)
	}
	a.connectivityData[conferenceID][roomID] = lang

	return nil
}

func (a *app) leave(conferenceID int, roomID string) error {
	_, ok := a.connectivityData[conferenceID]
	if !ok {
		return fmt.Errorf("conference %d doesn't exist", conferenceID)
	}
	delete(a.connectivityData[conferenceID], roomID)

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

		a.translateToJSON(req.Raw, language.Sinhala, w)
	}
}

func (a *app) translateToJSON(text string, lang language.Tag, w http.ResponseWriter) error {
	type respStruct struct {
		Raw string `json:"raw"`
	}

	respText, err := a.client.Translate(context.Background(), []string{text}, lang, nil)
	if err != nil {
		return fmt.Errorf("cannot get google api response: %v", err)
	}

	resp := respStruct{
		Raw: respText[0].Text,
	}

	respBody, err := json.Marshal(resp)
	if err != nil {
		if err != nil {
			return fmt.Errorf("cannot marshal response: %v", err)
		}
	}

	w.Write(respBody)
	return nil
}

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(w, r)
	})
}
