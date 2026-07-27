// internal/app/app.go

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"server-agent/internal/config"
	"server-agent/internal/monitor"
	"server-agent/internal/websocket"
)

type App struct {
	Config *config.Config
}

func New() (*App, error) {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return nil, err
	}

	token := os.Getenv("PANEL_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("PANEL_TOKEN env var is required")
	}

	cfg.Panel.Token = token

	if cfg.Agent.ID == "" {
		return nil, fmt.Errorf("agent ID is required in config")
	}

	return &App{Config: cfg}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Println("Server Agent started")
	log.Println("Panel URL:", a.Config.Panel.URL)
	log.Println("Agent ID:", a.Config.Agent.ID)

	// Инициализируем клиент ТОЛЬКО через New (без Dial)
	client := websocket.New(a.Config.Panel.URL)

	// Устанавливаем заголовки для WebSocket handshake
	client.SetHeader("X-Agent-Token", a.Config.Panel.Token)
	client.SetHeader("X-Agent-UUID", a.Config.Agent.ID)

	log.Println("Connecting to panel...")
	if err := client.Connect(ctx); err != nil {
		log.Printf("Connect failed: %v", err)
		return fmt.Errorf("failed to connect to panel: %w", err)
	}
	log.Println("Connected to panel")

	// Формируем payload для auth с обязательными полями
	hostname, hErr := os.Hostname()
	if hErr != nil {
		hostname = "unknown"
	}

	payloadData := map[string]any{
		"agent_id": a.Config.Agent.ID,
		"hostname": hostname,
		"os":       os.Getenv("GOOS"),
		"arch":     os.Getenv("GOARCH"),
	}

	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth payload: %w", err)
	}

	// Отправляем auth-сообщение
	err = client.Send(websocket.Message{
		Type:    "auth",
		Payload: payloadBytes,
	})
	if err != nil {
		log.Printf("Failed to send auth message: %v", err)
		client.Close()
		return fmt.Errorf("auth send failed: %w", err)
	}
	log.Println("Auth message sent")

	// Дальше твои рабочие циклы
	go a.loop(ctx, client)
	go a.readLoop(ctx, client)

	<-ctx.Done()

	log.Println("Stopping Server Agent")
	client.Close()
	return nil
}

func (a *App) loop(ctx context.Context, client *websocket.Client) {
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

			payload, mErr := json.Marshal(map[string]any{
				"agent_id": a.Config.Agent.ID,
				"stats":    stats,
			})
			if mErr != nil {
				log.Printf("marshal stats failed: %v", mErr)
				continue
			}

			msg := websocket.Message{
				Type:    "stats",
				Payload: payload,
			}

			if sErr := client.Send(msg); sErr != nil {
				log.Printf("send stats failed: %v", sErr)
				// соединение разорвано — можно выйти, переподключение лучше делать снаружи
				return
			}
		}
	}
}

func (a *App) readLoop(ctx context.Context, client *websocket.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := client.Read()
			if err != nil {
				log.Printf("read failed: %v", err)
				return // разрыв соединения
			}

			log.Printf("received: %s", string(data))

			var raw map[string]any
			if uErr := json.Unmarshal(data, &raw); uErr != nil {
				log.Printf("unmarshal failed: %v", uErr)
				continue
			}

			typ, ok := raw["type"].(string)
			if !ok {
				continue
			}

			switch typ {
			case "command.run":
				log.Printf("Command received: %+v", raw)
				// тут логика выполнения команды
			default:
				log.Printf("Unknown message type: %s", typ)
			}
		}
	}
}
