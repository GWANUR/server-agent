package websocket

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	url  string
}

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func New(url string) *Client {
	return &Client{url: url}
}

func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
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
