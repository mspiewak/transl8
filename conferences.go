package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

func (a *app) create(r room) int {
	if err := a.joinRoom(r.ID); err != nil {
		log.Println(err)
		return 0
	}
	ID := rand.Intn(100000)
	_, ok := a.connectivityData[ID]
	if ok {
		return a.create(r)
	}
	a.connectivityData[ID] = make(map[string]room)
	a.connectivityData[ID][r.ID] = r

	return ID
}

func (a *app) join(conferenceID int, r room) error {
	if err := a.joinRoom(r.ID); err != nil {
		return err
	}

	_, ok := a.connectivityData[conferenceID]
	if !ok {
		return fmt.Errorf("conference %d doesn't exist", conferenceID)
	}
	a.connectivityData[conferenceID][r.ID] = r

	return nil
}

func (a *app) leave(roomID string) {
	for conferenceID, v := range a.connectivityData {
		if _, ok := v[roomID]; ok {
			delete(a.connectivityData[conferenceID], roomID)
		}
	}
}

func resolveLanguage(command string) (language.Tag, error) {
	command = strings.TrimSpace(command)
	lastSpace := strings.LastIndex(command, " ") + 1
	return language.Parse(strings.TrimSpace(command[lastSpace:]))
}

func (a *app) routeRequest(req reqStruct) (string, error) {
	r := room{
		ID:   fmt.Sprintf("%s:%s", string(req.Source.Type[0]), req.Source.ID),
		Name: req.Source.Name,
	}
	switch true {
	case strings.Index(req.Raw, "@Transl8 create conference") == 0:
		fallthrough
	case strings.Index(req.Raw, "@Transl8 start conference") == 0:
		lang, err := resolveLanguage(req.Raw)
		if err != nil {
			return "Failed to create conference. Invalid language", err
		}
		r.Lang = lang
		r.ConferenceID = a.create(r)

		return fmt.Sprintf("Created conference ID: %d", r.ConferenceID), nil
	case strings.Index(req.Raw, "@Transl8 join conference") == 0:
		lang, err := resolveLanguage(req.Raw)
		if err != nil {
			return "", err
		}
		words := strings.Split(req.Raw, " ")
		conferenceID, err := strconv.Atoi(words[len(words)-2])
		if err != nil {
			return "", err
		}
		r.Lang = lang
		r.ConferenceID = conferenceID

		if err := a.join(conferenceID, r); err != nil {
			return "", err
		}
		return "Joined conference", nil
	case strings.Index(req.Raw, "@Transl8 leave conference") == 0:
		a.leave(r.ID)
		return "Left conference", nil
	}
	return `Message not understood.
Available commands:
*@Transl8* create conference {language code}
    Creates a conference and sets the language for the current room to the language
*@Transl8* join conference {conference id} {language code}
    Joins an existing conference and sets the language for the current room to the language"
*@Transl8* leave conference
    Removes the room from all registered conferences
    `, nil
}
