package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

func (a *app) create(roomID string, lang language.Tag) int {
	if err := a.joinRoom(roomID); err != nil {
		log.Println(err)
		return 0
	}
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
	if err := a.joinRoom(roomID); err != nil {
		return err
	}

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
	roomID := fmt.Sprintf("%s:%s", string(req.Source.Type[0]), req.Source.ID)
	switch true {
	case strings.Index(req.Raw, "@Transl8 create conference") == 0:
		fallthrough
	case strings.Index(req.Raw, "@Transl8 start conference") == 0:
		lang, err := resolveLanguage(req.Raw)
		if err != nil {
			return "Failed to create conference. Invalid language", err
		}
		confID := a.create(roomID, lang)

		return fmt.Sprintf("Created conference ID: %d", confID), nil
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

		err = a.join(conferenceID, roomID, lang)
		if err != nil {
			return "", err
		}
		return "Joined conference", nil
	case strings.Index(req.Raw, "@Transl8 leave conference") == 0:
		a.leave(roomID)
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
