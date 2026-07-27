package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn    *websocket.Conn
	headers http.Header
	url     string
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func New(url string) *Client {
	return &Client{
		url:     url,
		headers: http.Header{},
	}
}

func (c *Client) SetHeader(key, value string) {
	c.headers.Set(key, value)
}

func (c *Client) Connect(ctx context.Context) error {
	if c.conn != nil {
		return nil // уже подключено
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.Dial(c.url, c.headers)
	if err != nil {
		// Используем resp: если ошибка на уровне HTTP (например, 502 от Nginx),
		// resp будет не nil и покажет статус и тело ошибки.
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("WebSocket handshake failed: %v | HTTP status: %d | body: %s",
				err, resp.StatusCode, string(body))
		} else {
			log.Printf("WebSocket handshake failed: %v", err)
		}
		return err
	}

	c.conn = conn
	return nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Send(v any) error {
	if c.conn == nil {
		return fmt.Errorf("websocket not connected")
	}
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

func (c *Client) Read() ([]byte, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("websocket not connected")
	}
	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (c *Client) Ping() error {
	if c.conn == nil {
		return fmt.Errorf("websocket not connected")
	}
	return c.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second))
}
