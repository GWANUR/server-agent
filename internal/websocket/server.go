package websocket

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Serve(addr string, handler func(command string) (string, error)) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/agent", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			command := strings.TrimSpace(string(msg))
			if command == "" {
				continue
			}

			result, err := handler(command)
			if err != nil {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("ERROR: %v\n%s", err, result)))
				continue
			}
			_ = conn.WriteMessage(websocket.TextMessage, []byte(result))
		}
	})

	log.Printf("websocket server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}
