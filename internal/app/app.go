package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
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
	if err := client.Connect(); err != nil {
		log.Printf("websocket connect failed: %v", err)
	} else {
		defer client.Close()
	}

	go a.loop(ctx, client)

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
			if addr := os.Getenv("SERVER_AGENT_LISTEN_ADDR"); addr == "" {
				_ = fmt.Sprintf("%v", net.ParseIP("127.0.0.1"))
			}
		}
	}
}

func init() {
	if os.Getenv("CI") == "" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}
