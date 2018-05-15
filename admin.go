package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	adminPageTemplate = `
<html>
	<body>
		<h1>Transl8 Admin</h1>
		<table>
			<thead>
				<tr><th>Conference ID</th><th>Room Count</th><th></th></tr>
			</thead>
			<tbody>
				{{range .Conferences}}
				<tr><td>{{.ID}}</td><td>{{.RoomCount}}</td><td><a href="/details/{{.ID}}">See Conference Details</a></td></tr>
				{{end}}
			</tbody>
		</table>
	</body>
</html>
`
	detailsPageTemplate = `
<html>
	<body>
		<h1>Conference {{.ID}} Details</h1>
		<h2>Room List</h2>
		<table>
			<thead>
				<tr><th>Room ID</th><th>Room Name</th><th>Language</th></tr>
			</thead>
			<tbody>
				{{range .Rooms}}
				<tr><td>{{.ID}}</td><td>{{.Name}}</td><td>{{.Language}}</td></tr>
				{{end}}
			</tbody>
		</table>
	</body>
</html>
`
)

func (a *app) ParseTemplates() error {
	a.templates = make(map[string]*template.Template)
	admin, err := template.New("admin").Parse(adminPageTemplate)
	if err != nil {
		return fmt.Errorf("admin template failed to parse: %v", err)
	}
	a.templates["admin"] = admin

	details, err := template.New("details").Parse(detailsPageTemplate)
	if err != nil {
		return fmt.Errorf("details template failed to parse: %v", err)
	}
	a.templates["details"] = details
	return nil
}

func (a *app) adminHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		type conf struct {
			ID        int
			RoomCount int
		}
		var conferenceList []conf

		for i, rooms := range a.connectivityData {
			conferenceList = append(conferenceList, conf{i, len(rooms)})
		}

		a.templates["admin"].Execute(w, struct{ Conferences []conf }{conferenceList})
	}
}

func (a *app) detailsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		args := mux.Vars(r)
		type confRoom struct {
			ID       string
			Name     string
			Language string
		}
		type conf struct {
			Rooms []confRoom
		}
		conference := conf{}

		confID, err := strconv.Atoi(args["confID"])
		if err != nil {
			log.Printf("Invalid conf ID: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, rm := range a.connectivityData[confID] {
			conference.Rooms = append(conference.Rooms, confRoom{rm.ID, rm.Name, rm.Lang.String()})
		}

		a.templates["details"].Execute(w, conference)
	}
}
