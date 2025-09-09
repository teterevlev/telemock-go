package telemock

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	util "github.com/teterevlev/telemock-go/internal/util"
)

type clientPayload struct {
	ChatID       interface{} `json:"chat_id"`
	Text         string      `json:"text,omitempty"`
	MessageID    interface{} `json:"message_id,omitempty"`
	CallbackData string      `json:"callback_data,omitempty"`
}

type outboundPayload struct {
	ChatID           interface{}           `json:"chat_id"`
	Text             string                `json:"text"`
	From             string                `json:"from"`
	ReplyToMessageID interface{}           `json:"reply_to_message_id,omitempty"`
	IsReply          bool                  `json:"is_reply,omitempty"`
	MessageID        interface{}           `json:"message_id"`
	ReplyMarkup      *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

func (b *Bot) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		b.logger.Printf("telemock: upgrade failed: %v\n", err)
		return
	}
	b.addClient(conn)
	b.logger.Printf("telemock: client connected %s\n", conn.RemoteAddr())
	go b.readLoop(conn)
}

func (b *Bot) addClient(conn *websocket.Conn) {
	b.mu.Lock()
	b.clients[conn] = struct{}{}
	b.mu.Unlock()
}

func (b *Bot) removeClient(conn *websocket.Conn) {
	b.mu.Lock()
	delete(b.clients, conn)
	b.mu.Unlock()
	_ = conn.Close()
}

func (b *Bot) readLoop(conn *websocket.Conn) {
	defer func() {
		b.removeClient(conn)
		b.logger.Printf("telemock: client disconnected %s\n", conn.RemoteAddr())
	}()
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		// log incoming payload
		b.logRequest(conn.RemoteAddr().String(), raw)
		var cp clientPayload
		if err := json.Unmarshal(raw, &cp); err != nil {
			b.logger.Printf("telemock: invalid payload: %v\n", err)
			continue
		}
		if cp.CallbackData != "" {
			chatID, _ := util.ParseChatID(cp.ChatID)
			msgID := util.ParseToInt64(cp.MessageID)
			msg := &Message{
				MessageID: msgID,
				Chat:      Chat{ID: chatID},
				Text:      cp.Text,
				From:      &User{ID: chatID},
			}
			cq := &CallbackQuery{
				ID:      fmt.Sprintf("cb-%d-%d", time.Now().UnixNano(), msgID),
				From:    &User{ID: chatID},
				Message: msg,
				Data:    cp.CallbackData,
			}
			upd := Update{
				UpdateID:      atomic.AddInt64(&b.nextUpdID, 1),
				CallbackQuery: cq,
			}
			select {
			case b.updates <- upd:
			default:
				_ = b.drainOneUpdate()
				b.updates <- upd
			}
		} else if cp.Text != "" {
			chatID, _ := util.ParseChatID(cp.ChatID)
			msgID := util.ParseToInt64(cp.MessageID)
			if msgID == 0 {
				msgID = atomic.AddInt64(&b.nextMsgID, 1)
			}
			msg := &Message{
				MessageID: msgID,
				Chat:      Chat{ID: chatID},
				Text:      cp.Text,
				From:      &User{ID: chatID},
			}
			// Если начинается с /команда, добавим Entity типа bot_command до первого пробела
			if len(cp.Text) > 0 && cp.Text[0] == '/' {
				end := len(cp.Text)
				if sp := indexOfSpace(cp.Text); sp != -1 {
					end = sp
				}
				if end > 1 {
					msg.Entities = append(msg.Entities, MessageEntity{Type: "bot_command", Offset: 0, Length: end})
				}
			}
			upd := Update{
				UpdateID: atomic.AddInt64(&b.nextUpdID, 1),
				Message:  msg,
			}
			select {
			case b.updates <- upd:
			default:
				_ = b.drainOneUpdate()
				b.updates <- upd
			}
		} else {
			continue
		}
	}
}

// indexOfSpace returns index of first space or -1
func indexOfSpace(s string) int {
	return strings.IndexByte(s, ' ')
}

func (b *Bot) drainOneUpdate() error {
	select {
	case <-b.updates:
		return nil
	default:
		return errors.New("updates empty")
	}
}
