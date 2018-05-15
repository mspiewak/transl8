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
<head>
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css" integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/js/bootstrap.min.js" integrity="sha384-JZR6Spejh4U02d8jOt6vLEHfe/JQGiRRSQQxSfFWpi1MquVdAyjUar5+76PVCmYl" crossorigin="anonymous"></script>
<meta http-equiv="refresh" content="5">
</head>
	<body>
		<h1>Transl8 Admin</h1>
		<table class="table">
			<thead class="thead-dark">
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
<head>
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css" integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/js/bootstrap.min.js" integrity="sha384-JZR6Spejh4U02d8jOt6vLEHfe/JQGiRRSQQxSfFWpi1MquVdAyjUar5+76PVCmYl" crossorigin="anonymous"></script>
<meta http-equiv="refresh" content="5">
</head>
	<body>
		<h1>Conference {{.ID}} Details</h1>
		<h2>Room List</h2>
		<table class="table">
			<thead class="thead-dark">
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
			ID    int
			Rooms []confRoom
		}

		confID, err := strconv.Atoi(args["confID"])
		if err != nil {
			log.Printf("Invalid conf ID: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		conference := conf{
			ID: confID,
		}

		for _, rm := range a.connectivityData[confID] {
			conference.Rooms = append(conference.Rooms, confRoom{rm.ID, rm.Name, rm.Lang.String()})
		}

		a.templates["details"].Execute(w, conference)
	}
}
