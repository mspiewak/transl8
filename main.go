package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"cloud.google.com/go/translate"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

type app struct {
	client           *translate.Client
	connectivityData map[int]map[string]room
	ctx              context.Context
	wsConn           *websocket.Conn
	sessionKey       string
}

type room struct {
	ID           string
	Name         string
	Lang         language.Tag
	ConferenceID int
}

func main() {
	var gAPIKey string
	var chaletBotKey string
	var chaletURL string
	flag.StringVar(&gAPIKey, "gApiKey", "", "google api key")
	flag.StringVar(&chaletBotKey, "chaletBotKey", "", "chalet bot key")
	flag.StringVar(&chaletURL, "chaletUrl", "api.us-east.chalet.8x8.com", "chalet url")
	flag.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: chaletURL, Path: "/ws/v1"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	var app app
	app.connectivityData = make(map[int]map[string]room)
	app.ctx = context.Background()
	app.client, err = translate.NewClient(app.ctx, option.WithAPIKey(gAPIKey))
	if err != nil {
		log.Fatal(err)
	}
	app.wsConn = c

	authRequest := []authRequest{{
		UUID: uuid.NewV4().String(),
		Data: authRequestData{
			Type:  "hello",
			Token: chaletBotKey,
		},
	}}

	if err := c.WriteJSON(authRequest); err != nil {
		log.Fatalf("cannot register ws: %v", err)
		return
	}

	var a auth
	if err := c.ReadJSON(&a); err != nil {
		log.Fatalf("cannot read session: %v", err)
	}

	app.sessionKey = a[0].Data.SessionKey

	done := make(chan struct{})
	rand.Seed(time.Now().UTC().UnixNano())

	go app.wsHandler(done)

	r := mux.NewRouter()
	r.Handle("/transl8", app.transl8Handler()).Methods(http.MethodPost)

	log.Println("listening")
	go http.ListenAndServe(":9010", commonHeaders(r))

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			p := []ping{
				{
					socketRequest: getSocketRequest(app.sessionKey),
					Data:          "ping",
				},
			}
			if err := c.WriteJSON(p); err != nil {
				log.Println("ping error:", err)
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
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
	TS        int64  `json:"ts"`
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

func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		next.ServeHTTP(w, r)
	})
}
