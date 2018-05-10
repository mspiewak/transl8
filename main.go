package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type app struct{}

func main() {
	var app app
	r := mux.NewRouter()
	r.Handle("/translate", app.translateHandler()).Methods(http.MethodPost)

	log.Println("listening")
	log.Fatal(http.ListenAndServe(":9010", commonHeaders(r)))
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
		resp := respStruct{
			Raw: req.Raw,
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
