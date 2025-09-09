package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type scenarioLine struct {
	ChatID       int64           `json:"chat_id"`
	Text         string          `json:"text"`
	From         string          `json:"from"`
	MessageID    int64           `json:"message_id"`
	CallbackData string          `json:"callback_data"`
	ReplyMarkup  json.RawMessage `json:"reply_markup"`
}

func TestScenario_Simple(t *testing.T) {
	// 1) Запускаем реальный бот-процесс из каталога примера
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = "."
	cmd.Stdout = nil // игнорируем вывод
	cmd.Stderr = nil // игнорируем вывод
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start bot process: %v", err)
	}

	// Завершаем процесс корректно через os.Interrupt, затем Kill если нужно
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt) // мягкий сигнал
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-time.After(2 * time.Second):
				_ = cmd.Process.Kill() // принудительно, если не завершился
			case <-done:
			}
		}
	}()

	// 2) Ждем, пока WS поднимется, и подключаемся
	d := websocket.Dialer{}
	var ws *websocket.Conn
	var err error
	for i := 0; i < 40; i++ { // ~2s
		ws, _, err = d.Dial("ws://localhost:8765/", nil)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to connect to telemock ws: %v", err)
	}
	defer ws.Close()

	// 3) Открываем сценарий (относительно этого пакета)
	f, err := os.Open("../testdata/scenarios/simple.jsonl")
	if err != nil {
		t.Fatalf("failed to open scenario: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var sl scenarioLine
		if err := json.Unmarshal([]byte(line), &sl); err != nil {
			t.Fatalf("malformed jsonl line: %v", err)
		}

		if sl.From == "bot" {
			// Ждем ответ бота и сравниваем
			_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, b, err := ws.ReadMessage()
			if err != nil {
				t.Fatalf("expected bot message, read error: %v", err)
			}

			var got scenarioLine
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("bot message unmarshal error: %v; raw: %s", err, string(b))
			}

			if got.ChatID != sl.ChatID || got.Text != sl.Text || got.From != "bot" {
				t.Fatalf("unexpected bot message: got chat_id=%d text=%q from=%q; want chat_id=%d text=%q from=bot",
					got.ChatID, got.Text, got.From, sl.ChatID, sl.Text)
			}

			if len(sl.ReplyMarkup) > 0 {
				if len(got.ReplyMarkup) == 0 {
					t.Fatalf("expected reply_markup, but none present in bot message")
				}
				var wantAny any
				var gotAny any
				if err := json.Unmarshal(sl.ReplyMarkup, &wantAny); err != nil {
					t.Fatalf("invalid expected reply_markup: %v", err)
				}
				if err := json.Unmarshal(got.ReplyMarkup, &gotAny); err != nil {
					t.Fatalf("invalid got reply_markup: %v", err)
				}
				wantBytes, _ := json.Marshal(wantAny)
				gotBytes, _ := json.Marshal(gotAny)
				if string(wantBytes) != string(gotBytes) {
					t.Fatalf("reply_markup mismatch:\nwant: %s\n got: %s", string(wantBytes), string(gotBytes))
				}
			}
			continue
		}

		// Исходящее в бот
		_ = ws.SetWriteDeadline(time.Now().Add(2 * time.Second))
		if err := ws.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			t.Fatalf("failed to send inbound message: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
}
