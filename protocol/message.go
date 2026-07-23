package protocol

import "encoding/json"

type Message struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type AuthPayload struct {
	AgentID string `json:"agent_id"`
	Token   string `json:"token"`
}

type HeartbeatPayload struct {
	Timestamp int64 `json:"timestamp"`
}

type ExecutePayload struct {
	Command string `json:"command"`
}

type TerminalOutputPayload struct {
	Data string `json:"data"`
}

type ServerInfoPayload struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Kernel   string `json:"kernel"`
	CPU      string `json:"cpu"`
	Cores    int    `json:"cores"`
	RAM      uint64 `json:"ram"`
}

type MetricsPayload struct {
	CPUUsage float64 `json:"cpu_usage"`
	RAMUsage float64 `json:"ram_usage"`

	DiskUsage float64 `json:"disk_usage"`

	NetworkIn uint64 `json:"network_in"`
	NetworkOut uint64 `json:"network_out"`

	Uptime uint64 `json:"uptime"`
}

func Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}