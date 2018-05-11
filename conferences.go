package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

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
