package app

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"server-agent/internal/config"
	"server-agent/internal/monitor"
	"server-agent/internal/websocket"
	"time"
)

type App struct {
	Config *config.Config
}

func New() (*App, error) {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		return nil, err
	}
	return &App{Config: cfg}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Println("Server Agent started")
	log.Println("Panel URL:", a.Config.Panel.URL)
	log.Println("Agent ID:", a.Config.Agent.ID)

	client := websocket.New(a.Config.Panel.URL)

	for {
		log.Println("Connecting to panel...")

		if err := client.Connect(); err == nil {
			log.Println("Connected to panel")
			payload, _ := json.Marshal(map[string]any{
				"agent_id": a.Config.Agent.ID,
				"test":     true,
			})

			err := client.Send(websocket.Message{
				Type:    "hello",
				Payload: payload,
			})

			log.Printf("Send result: %v", err)
			break
		} else {
			log.Printf("Connect failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
	}

	defer client.Close()

	go a.loop(ctx, client)
	go a.readLoop(ctx, client)

	<-ctx.Done()

	log.Println("Stopping Server Agent")
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
			if client != nil {
				payload, _ := json.Marshal(map[string]any{"agent_id": a.Config.Agent.ID, "stats": stats})
				_ = client.Send(websocket.Message{Type: "stats", Payload: payload})
			}
		}
	}
}

func init() {
	if os.Getenv("CI") == "" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

func (a *App) readLoop(ctx context.Context, client *websocket.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := client.Read()
			if err != nil {
				log.Printf("read failed: %v", err)
				return
			}

			log.Printf("received: %s", msg)

			// Здесь обработка команд от панели
		}
	}
}
