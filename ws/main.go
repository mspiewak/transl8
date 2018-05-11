package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
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

type messages []struct {
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

type postMessageData struct {
	Type         string `json:"type"`
	RoomID       string `json:"room_id"`
	Raw          string `json:"raw"`
	BotName      string `json:"bot_name"`
	BotEmail     string `json:"bot_email"`
	BotAvatarURL string `json:"bot_avatar_url"`
}

type postMessage struct {
	socketRequest
	Data postMessageData `json:"data"`
}

type ackMessage []struct {
	AckUuids []string `json:"ack_uuids"`
	Data     string   `json:"data"`
}

var addr = flag.String("addr", "api.us-east.chalet.8x8.com", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "wss", Host: *addr, Path: "/ws/v1"}
	log.Printf("connecting to %s", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	jsonText := `[{"uuid":"83519310-550c-11e8-9f4f-674c4437bc21","data":{"type":"hello","token":"eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiI4MTc4MzIwNDkyNzQwNDA0OTM3ODI0MTkyOTc0NTEiLCJjc3QiOiJhZG1pbiIsImxpZCI6IjhYOEdCLjQ3NDciLCJpc3MiOiJodHRwczovL3BsYXRmb3JtLjh4OC5jb20iLCJ0eXAiOiJhY2Nlc3MiLCJhdWQiOiJ2b20iLCJzY2QiOiJ1c2VyIiwiZXhwIjoxNTI3MjQyODY1LCJpYXQiOjE1MjYwMzMyNjUsImJybyI6ZmFsc2UsInNjbyI6IjB4ODQiLCJqdGkiOiI1MzMwNGMzZS1mN2Q2LTQ1YmEtYjJmMy1lN2MxNzNkMzJhYjIiLCJjaWQiOiIyNTI1MDk3NzA2MjExMzU5Nzc4ODQ1ODkyOTQwMzIifQ.n3jfPswBsGHT8SEi2UFuak-xO_bt47WalW0gf26I85OWPk47kRbuvMWmr7SnIHQZLBkIyBx9BVVswk4YTJBjbsqP3XvqsOFq8O1tLa0e2n7GpGd_KL5yDs-sjJccXB1Jc3g6WrrHsDlT_Vgc1-4uh-vY5nQlhTJFCZDqpGf3i4z6oapyUAFptdLDfrDH_l2jFbpHtjXIDfI9rWeOUlKGcQJtbMU8vAltcYbva-fILyrn1bFFHQs1pecueTbBRRV25ft1W3NMrQSd8sZ1XUJ_lw33BZ3Y7744h8IWXUM4nSlwvIqVgx4cWXE807Llgr-Ta9Jde5haV8fBIQJuZpGK5g"}}]`
	if err := c.WriteMessage(websocket.TextMessage, []byte(jsonText)); err != nil {
		log.Println("write:", err)
		return
	}

	var a auth
	if err := c.ReadJSON(&a); err != nil {
		log.Fatalf("read: %v", err)
	}

	done := make(chan struct{})

	j := []join{
		{
			socketRequest: getSocketRequest(a[0].Data.SessionKey),
			Data: joinData{
				Type:   "join",
				RoomID: "d:NzU0MTQ2Nzg4Mzk5MDMxMjYxNjk0Mjk0MTkzODQ4:ODE3ODMyMDQ5Mjc0MDQwNDkzNzgyNDE5Mjk3NDUx",
			},
		},
	}
	if err := c.WriteJSON(j); err != nil {
		log.Println("write:", err)
		return
	}

	defer c.Close()

	go func() {
		defer close(done)
		type uuidAck []struct {
			UUID string `json:"uuid"`
		}

		for {
			_, bodymsg, err := c.ReadMessage()
			if err != nil {
				fmt.Println(err)
				continue
			}
			var uid uuidAck
			if err := json.Unmarshal(bodymsg, &uid); err != nil {
				fmt.Printf("cannot unmarshal json: %v\n", err)
				continue
			}
			if err := c.WriteJSON(ackMessage{{AckUuids: []string{uid[0].UUID}, Data: "ack"}}); err != nil {
				log.Printf("cannot acknowledge message %s: %v", uid[0].UUID, err)
				continue
			}

			var m messages
			if err := json.Unmarshal(bodymsg, &m); err != nil {
				continue
			}

			if m[0].Data.Type != "text_parsed" || m[0].Data.AuthorUser.Email == "marcin.spiewak+lama@8x8.com" {
				continue
			}

			p := []postMessage{
				{
					socketRequest: getSocketRequest(a[0].Data.SessionKey),
					Data: postMessageData{
						BotAvatarURL: "https://media.istockphoto.com/vectors/lama-alpaca-sad-animal-sorrowful-icon-vector-illustration-vector-id874676612?s=170x170",
						BotEmail:     "marcin.spiewak+lama@8x8.com",
						BotName:      "Lama transl8",
						Raw:          m[0].Data.Parsed,
						RoomID:       "d:NzU0MTQ2Nzg4Mzk5MDMxMjYxNjk0Mjk0MTkzODQ4:ODE3ODMyMDQ5Mjc0MDQwNDkzNzgyNDE5Mjk3NDUx",
						Type:         "text_raw",
					},
				},
			}

			if err := c.WriteJSON(p); err != nil {
				log.Println("cannot post message: %v", err)
			}
		}
	}()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			p := []ping{
				{
					socketRequest: getSocketRequest(a[0].Data.SessionKey),
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

func getSocketRequest(sessionKey string) socketRequest {
	return socketRequest{
		Ts:           time.Now().Unix(),
		UUID:         uuid.Must(uuid.NewV4()).String(),
		RetryPolicy:  "same_session",
		TimedOutAtTs: time.Now().Unix() + int64(1000000),
		SessionKey:   sessionKey,
	}
}
