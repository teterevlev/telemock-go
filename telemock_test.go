package telemock

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// dialWS retries websocket dial until success or timeout
func dialWS(t *testing.T) *websocket.Conn {
	deadline := time.Now().Add(2 * time.Second)
	var conn *websocket.Conn
	var err error
	for time.Now().Before(deadline) {
		u := url.URL{Scheme: "ws", Host: "127.0.0.1:8765", Path: "/"}
		conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err == nil {
			return conn
		}
		time.Sleep(25 * time.Millisecond)
	}
	require.NoError(t, err)
	return nil
}

func TestSendMessage_ReplyFieldsAndDelivery(t *testing.T) {
	bot, err := NewBot("token")
	require.NoError(t, err)
	defer bot.Close(context.Background())

	conn := dialWS(t)
	defer conn.Close()

	params := &SendMessageParams{
		ChatID:           ChatID{ID: 123},
		Text:             "Hello",
		ReplyToMessageID: 42,
	}
	_, err = bot.SendMessage(context.Background(), params)
	require.NoError(t, err)

	_, raw, err := conn.ReadMessage()
	require.NoError(t, err)

	var out outboundPayload
	require.NoError(t, json.Unmarshal(raw, &out))
	// numbers in interface{} become float64 after json.Unmarshal
	require.Equal(t, float64(123), out.ChatID)
	require.Equal(t, "Hello", out.Text)
	require.Equal(t, true, out.IsReply)
	require.Equal(t, float64(42), out.ReplyToMessageID)
}

func TestUpdatesViaLongPolling_TextMessage(t *testing.T) {
	bot, err := NewBot("token")
	require.NoError(t, err)
	defer bot.Close(context.Background())

	conn := dialWS(t)
	defer conn.Close()

	updates, err := bot.UpdatesViaLongPolling(context.Background(), &GetUpdatesParams{})
	require.NoError(t, err)

	cp := clientPayload{ChatID: 777, Text: "ping", MessageID: 1}
	data, _ := json.Marshal(cp)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))

	select {
	case upd := <-updates:
		require.NotNil(t, upd.Message)
		require.Equal(t, int64(777), upd.Message.Chat.ID)
		require.Equal(t, "ping", upd.Message.Text)
		return
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for update")
	}
}

func TestUpdatesViaLongPolling_Callback(t *testing.T) {
	bot, err := NewBot("token")
	require.NoError(t, err)
	defer bot.Close(context.Background())

	conn := dialWS(t)
	defer conn.Close()

	updates, err := bot.UpdatesViaLongPolling(context.Background(), &GetUpdatesParams{})
	require.NoError(t, err)

	cp := clientPayload{ChatID: 555, MessageID: 10, CallbackData: "ok", Text: "btn"}
	data, _ := json.Marshal(cp)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, data))

	select {
	case upd := <-updates:
		require.NotNil(t, upd.CallbackQuery)
		require.Equal(t, int64(555), upd.CallbackQuery.From.ID)
		require.Equal(t, "ok", upd.CallbackQuery.Data)
		return
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for callback update")
	}
}
