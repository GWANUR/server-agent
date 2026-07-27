package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"server-agent/internal/config"
	"server-agent/internal/monitor"

	"github.com/gorilla/websocket" // или твой internal/websocket, но важно, чтобы он использовал gorilla/websocket
)

type App struct {
	Config *config.Config
}

func New() (*App, error) {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return nil, err
	}
	// Проверка, что токен и UUID есть
	if cfg.Panel.Token == "" || cfg.Agent.ID == "" {
		return nil, fmt.Errorf("missing token or agent ID in config")
	}
	return &App{Config: cfg}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Println("Server Agent started")
	log.Println("Panel URL:", a.Config.Panel.URL)
	log.Println("Agent ID:", a.Config.Agent.ID)

	// Подготовка заголовков для WebSocket handshake
	headers := http.Header{}
	headers.Set("X-Agent-Token", a.Config.Panel.Token)
	headers.Set("X-Agent-UUID", a.Config.Agent.ID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(a.Config.Panel.URL, headers)
	if err != nil {
		log.Printf("Dial failed: %v", err)
		if resp != nil {
			log.Printf("HTTP status: %d, body: %s", resp.StatusCode, resp.Body)
		}
		return err
	}
	defer conn.Close()

	log.Println("Connected to panel (handshake successful)")

	// Здесь можно отправить первое сообщение, если сервер требует, например, "ready"
	// Но НЕ отправляй отдельный auth-пакет, если аутентификация уже была в заголовках

	go a.loop(ctx, conn)
	go a.readLoop(ctx, conn)

	<-ctx.Done()
	log.Println("Stopping Server Agent")
	return nil
}

func (a *App) loop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(time.Duration(a.Config.Monitor.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := monitor.Collect()
			if err != nil {
				log.Printf("collect metrics failed: %v", err)
				continue
			}

			payload, err := json.Marshal(map[string]any{
				"agent_id": a.Config.Agent.ID,
				"stats":    stats,
			})
			if err != nil {
				log.Printf("marshal stats failed: %v", err)
				continue
			}

			message := map[string]any{
				"type":    "stats",
				"payload": payload,
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("marshal message failed: %v", err)
				continue
			}

			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Printf("write stats failed: %v", err)
				return // соединение разорвано, нужно переподключаться
			}
		}
	}
}

func (a *App) readLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read failed: %v", err)
				return // разрыв соединения, Run завершит работу, можно сделать retry снаружи
			}

			log.Printf("received: %s", msg)

			// Тут парсишь команды от панели и выполняешь
			var raw map[string]any
			if err := json.Unmarshal(msg, &raw); err != nil {
				log.Printf("unmarshal failed: %v", err)
				continue
			}

			typ, ok := raw["type"].(string)
			if !ok {
				continue
			}

			switch typ {
			case "command.run":
				// тут логика выполнения команды
				log.Printf("Command received: %+v", raw)
			default:
				log.Printf("Unknown message type: %s", typ)
			}
		}
	}
}
