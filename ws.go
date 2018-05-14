package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	uuid "github.com/satori/go.uuid"
)

type auth []struct {
	AckUuids []string `json:"ack_uuids"`
	Data     struct {
		Type      string `json:"type"`
		SipConfig struct {
			WsServers         []string `json:"wsServers"`
			URI               string   `json:"uri"`
			Password          string   `json:"password"`
			AuthorizationUser string   `json:"authorizationUser"`
		} `json:"sip_config"`
		SessionKey       string `json:"session_key"`
		ServerVersion    string `json:"server_version"`
		PublishedVersion string `json:"published_version"`
		Limits           struct {
			MaxUploadsSizeBytes   int `json:"max_uploads_size_bytes"`
			MaxRoomPageSize       int `json:"max_room_page_size"`
			MaxForumNameLength    int `json:"max_forum_name_length"`
			FixedRoomJoinPageSize int `json:"fixed_room_join_page_size"`
			DefaultRoomPageSize   int `json:"default_room_page_size"`
		} `json:"limits"`
	} `json:"data"`
}

type authRequestData struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type authRequest struct {
	UUID string          `json:"uuid"`
	Data authRequestData `json:"data"`
}

type messagesGeneric []struct {
	UUID string                 `json:"uuid"`
	Data map[string]interface{} `json:"data"`
}

type socketRequest struct {
	Ts           int64  `json:"ts"`
	UUID         string `json:"uuid"`
	RetryPolicy  string `json:"retry_policy"`
	TimedOutAtTs int64  `json:"timed_out_at_ts"`
	SessionKey   string `json:"session_key"`
}

type joinData struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
}

type join struct {
	socketRequest
	Data joinData `json:"data"`
}

type ping struct {
	socketRequest
	Data string `json:"data"`
}

type postMessage struct {
	socketRequest
	Data postMessageData `json:"data"`
}

type ackMessage []struct {
	AckUuids []string `json:"ack_uuids"`
	Data     string   `json:"data"`
}

func getSocketRequest(sessionKey string) socketRequest {
	return socketRequest{
		Ts:           time.Now().Unix(),
		UUID:         uuid.Must(uuid.NewV4()).String(),
		RetryPolicy:  "same_session",
		TimedOutAtTs: time.Now().Unix() + int64(1000000),
		SessionKey:   sessionKey,
	}
}

func (app *app) joinRoom(roomID string) error {
	j := []join{
		{
			socketRequest: getSocketRequest(app.sessionKey),
			Data: joinData{
				Type:   "join",
				RoomID: roomID,
			},
		},
	}
	if err := app.wsConn.WriteJSON(j); err != nil {
		return fmt.Errorf("cannot join room: %v", err)
	}

	return nil
}

func (app *app) wsHandler(done chan struct{}) {
	defer close(done)
	type uuidAck []struct {
		UUID string `json:"uuid"`
	}

	for {
		_, bodymsg, err := app.wsConn.ReadMessage()
		if err != nil {
			log.Println(err)
			continue
		}

		var uid uuidAck
		if err := json.Unmarshal(bodymsg, &uid); err != nil {
			log.Printf("cannot unmarshal json: %v\n", err)
			continue
		}
		if err := app.wsConn.WriteJSON(ackMessage{{AckUuids: []string{uid[0].UUID}, Data: "ack"}}); err != nil {
			log.Printf("cannot acknowledge message %s: %v", uid[0].UUID, err)
			continue
		}

		var m []message
		if err := json.Unmarshal(bodymsg, &m); err != nil {
			continue
		}

		if m[0].Data.Type != "text_parsed" || m[0].Data.AuthorUser.Email == "transl8" {
			continue
		}

		app.ShareMessage(m[0])
	}
}
