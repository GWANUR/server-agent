package terminal

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type Session struct {
	ID      string
	Command string
	Output  string
}

func Execute(command string) (*Session, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	result := &Session{Command: command, Output: string(output)}
	if err != nil {
		return result, err
	}
	return result, nil
}
