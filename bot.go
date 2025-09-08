package telemock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Bot struct {
	addr       string
	upgrader   websocket.Upgrader
	mu         sync.RWMutex
	clients    map[*websocket.Conn]struct{}
	updates    chan Update
	nextUpdID  int64
	nextMsgID  int64
	httpServer *http.Server
	listener   net.Listener
	closed     chan struct{}
	logger     *log.Logger
}

// NewBot starts telemock WS server listening on default address ":8765".
// token is ignored but kept for API compatibility.
func NewBot(token string) (*Bot, error) {
	addr := ":8765"

	b := &Bot{
		addr:     addr,
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:  make(map[*websocket.Conn]struct{}),
		updates:  make(chan Update, 256),
		closed:   make(chan struct{}),
		logger:   log.Default(),
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	b.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/", b.handleWS)

	srv := &http.Server{
		Handler: mux,
	}
	b.httpServer = srv

	go func() {
		b.logger.Printf("telemock: WebSocket server starting on %s\n", addr)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			b.logger.Printf("telemock: server error: %v\n", err)
		}
		close(b.closed)
	}()

	return b, nil
}

func (b *Bot) UpdatesViaLongPolling(ctx context.Context, _ *GetUpdatesParams) (<-chan Update, error) {
	out := make(chan Update)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case u, ok := <-b.updates:
				if !ok {
					return
				}
				select {
				case out <- u:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func (b *Bot) SendMessage(ctx context.Context, params *SendMessageParams) (*Message, error) {
	if params == nil {
		return nil, errors.New("nil params")
	}
	msgID := atomic.AddInt64(&b.nextMsgID, 1)
	message := &Message{
		MessageID: msgID,
		Chat:      Chat{ID: params.ChatID.ID},
		Text:      params.Text,
		From:      &User{ID: 0, Name: "bot"},
	}

	out := outboundPayload{
		ChatID:           params.ChatID.ID,
		Text:             params.Text,
		From:             "bot",
		MessageID:        msgID,
		ReplyToMessageID: params.ReplyToMessageID,
	}

	if params.ReplyMarkup != nil {
		out.ReplyMarkup = params.ReplyMarkup
	}

	if params.ReplyToMessageID != 0 {
		out.IsReply = true
	}

	b.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(b.clients))
	for c := range b.clients {
		conns = append(conns, c)
	}
	b.mu.RUnlock()

	if len(conns) == 0 {
		return message, nil
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}

	for _, c := range conns {
		_ = c.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			b.logger.Printf("telemock: write error: %v\n", err)
			go b.removeClient(c)
		}
	}

	return message, nil
}

func (b *Bot) AnswerCallbackQuery(ctx context.Context, callbackID string, text string) error {
	_ = callbackID
	_ = text
	return nil
}

func (b *Bot) Close(ctx context.Context) error {
	if b.httpServer != nil {
		_ = b.httpServer.Shutdown(ctx)
	}
	// close all websockets
	b.mu.Lock()
	for c := range b.clients {
		_ = c.Close()
	}
	b.clients = map[*websocket.Conn]struct{}{}
	b.mu.Unlock()

	// close updates channel
	close(b.updates)

	// wait for serve goroutine to finish
	select {
	case <-b.closed:
	case <-time.After(2 * time.Second):
	}

	return nil
}
