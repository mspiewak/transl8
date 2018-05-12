package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
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
	r.Handle("/", app.transl8Handler()).Methods(http.MethodPost)

	log.Println("listening")
	log.Fatal(http.ListenAndServe("", commonHeaders(r)))
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

func (a *app) leave(roomID string) {

	for conferenceID, v := range a.connectivityData {
		if _, ok := v[roomID]; ok {
			delete(a.connectivityData[conferenceID], roomID)
		}
	}
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
	TS        string `json:"ts"`
	Raw       string `json:"raw"`
}

type respStruct struct {
	Raw string `json:"raw"`
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

func resolveLanguage(command string) (language.Tag, error) {
	command = strings.TrimSpace(command)
	lastSpace := strings.LastIndex(command, " ") + 1
	return language.Parse(strings.TrimSpace(command[lastSpace:]))
}

func (a *app) routeRequest(req reqStruct) (string, error) {
	roomID := fmt.Sprintf("%s:%s:%s", req.OrgID, req.Source.Type, req.Source.ID)
	switch true {
	case strings.Index(req.Raw, "@transl8 create conference") == 0:
		fallthrough
	case strings.Index(req.Raw, "@transl8 start conference") == 0:
		lang, err := resolveLanguage(req.Raw)
		if err != nil {
			return "Failed to create conference. Invalid language", err
		}
		confID := a.create(roomID, lang)

		return fmt.Sprintf("Created conference ID: %d", confID), nil
	case strings.Index(req.Raw, "@transl8 join conference") == 0:
		lang, err := resolveLanguage(req.Raw)
		if err != nil {
			return "", err
		}
		words := strings.Split(req.Raw, " ")
		conferenceID, err := strconv.Atoi(words[len(words)-2])
		if err != nil {
			return "", err
		}

		err = a.join(conferenceID, roomID, lang)
		if err != nil {
			return "", err
		}
		return "Joined conference", nil
	case strings.Index(req.Raw, "@transl8 leave conference") == 0:
		a.leave(roomID)
		return "Left conference", nil
	}
	return `Message not understood.
Available commands:
<b>@transl8 create conference {language code}</b> Creates a conference and sets the language for the current room to the language"
<b>@transl8 join conference {conference id} {language code}</b> Joins an existing conference and sets the language for the current room to the language"
<b>@transl8 leave conference</b> Removes the room from all registered conferences`, nil
}

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(w, r)
	})
}
