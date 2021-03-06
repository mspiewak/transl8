package main

import (
	"fmt"
	"html"
	"log"
	"regexp"

	"cloud.google.com/go/translate"

	"golang.org/x/text/language"
)

type postMessageData struct {
	Type         string `json:"type"`
	RoomID       string `json:"room_id"`
	Raw          string `json:"raw"`
	BotName      string `json:"bot_name"`
	BotEmail     string `json:"bot_email"`
	BotAvatarURL string `json:"bot_avatar_url"`
}

type message struct {
	UUID string `json:"uuid"`
	Data struct {
		Type       string `json:"type"`
		Ts         int64  `json:"ts"`
		RoomID     string `json:"room_id"`
		Parsed     string `json:"parsed"`
		ID         string `json:"id"`
		AuthorUser struct {
			Name      string `json:"name"`
			ID        string `json:"id"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar_url"`
		} `json:"author_user"`
	} `json:"data"`
}

type MessagePoster interface {
	PostMessage(data postMessageData) error
}

func (a *app) PostMessage(data postMessageData) error {
	p := []postMessage{
		{
			socketRequest: getSocketRequest(a.sessionKey),
			Data:          data,
		},
	}

	if err := a.wsConn.WriteJSON(p); err != nil {
		return err
	}
	return nil
}

func (a *app) ShareMessage(m message) error {
	confs := a.findConferencesForRoom(m.Data.RoomID)

	translations := make(map[language.Tag]string)

	detecteds, err := a.client.DetectLanguage(a.ctx, []string{m.Data.Parsed})
	if err != nil {
		return fmt.Errorf("failed to detect language: %v", err)
	}

	translations[detecteds[0][0].Language] = m.Data.Parsed

	for _, c := range confs {
		for r, room := range a.connectivityData[c] {
			if detecteds[0][0].Language == room.Lang && r == m.Data.RoomID {
				continue
			}
			if _, ok := translations[room.Lang]; ok == false {
				trans, err := a.client.Translate(a.ctx, []string{m.Data.Parsed}, room.Lang, &translate.Options{})
				if err != nil {
					return fmt.Errorf("failed to translate to %s: %v", room.Lang, err)
				}
				translations[room.Lang] = trans[0].Text
				log.Println(translations[room.Lang])
			}
			err = a.PostMessage(
				postMessageData{
					Type:         "text_raw",
					RoomID:       r,
					Raw:          parseMessage(translations[room.Lang]),
					BotName:      m.Data.AuthorUser.Name,
					BotEmail:     "transl8",
					BotAvatarURL: m.Data.AuthorUser.AvatarURL,
				},
			)
		}
	}
	return nil
}

func parseMessage(msg string) string {
	boldTag := regexp.MustCompile(`<(/?)b\b((?:[^>"']|"[^"]*"|'[^']*')*)>`)
	return boldTag.ReplaceAllString(html.UnescapeString(msg), "*")
}

func (a *app) findConferencesForRoom(roomID string) []int {
	conferences := []int{}
	for c, rooms := range a.connectivityData {
		for r, _ := range rooms {
			if r == roomID {
				conferences = append(conferences, c)
			}
		}
	}
	return conferences
}
