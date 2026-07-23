package monitor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Stats struct {
	Timestamp time.Time `json:"timestamp"`
	Hostname  string    `json:"hostname"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	CPU       CPUStats  `json:"cpu"`
	Memory    MemoryStats `json:"memory"`
	Disk      DiskStats `json:"disk"`
	Load      LoadStats `json:"load"`
	Uptime    string    `json:"uptime"`
	Logs      []LogEntry `json:"logs"`
}

type CPUStats struct {
	Usage float64 `json:"usage"`
	Count int     `json:"count"`
}

type MemoryStats struct {
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	Percent float64 `json:"percent"`
}

type DiskStats struct {
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	Percent float64 `json:"percent"`
}

type LoadStats struct {
	OneMinute     float64 `json:"one_minute"`
	FiveMinutes   float64 `json:"five_minutes"`
	FifteenMinutes float64 `json:"fifteen_minutes"`
}

type LogEntry struct {
	Source    string `json:"source"`
	Line      string `json:"line"`
	Timestamp string `json:"timestamp"`
}

func Collect() (*Stats, error) {
	stats := &Stats{
		Timestamp: time.Now().UTC(),
		Hostname:  hostname(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	cpuUsage, err := cpuUsage()
	if err != nil {
		return nil, err
	}
	stats.CPU = CPUStats{Usage: cpuUsage, Count: runtime.NumCPU()}

	mem, err := memoryStats()
	if err != nil {
		return nil, err
	}
	stats.Memory = mem

	disk, err := diskStats()
	if err != nil {
		return nil, err
	}
	stats.Disk = disk

	load, err := loadStats()
	if err != nil {
		return nil, err
	}
	stats.Load = load

	stats.Uptime = uptime()
	stats.Logs = recentLogs()

	return stats, nil
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func cpuUsage() (float64, error) {
	if runtime.GOOS != "linux" {
		return 0, nil
	}

	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			parts := strings.Fields(line)
			if len(parts) < 8 {
				return 0, fmt.Errorf("unexpected /proc/stat format")
			}
			values := make([]uint64, 0, 8)
			for _, part := range parts[1:] {
				v, err := strconv.ParseUint(part, 10, 64)
				if err != nil {
					return 0, err
				}
				values = append(values, v)
			}
			idle := values[3] + values[4]
			total := uint64(0)
			for _, value := range values {
				total += value
			}
			if total == 0 {
				return 0, nil
			}
			return round((1 - float64(idle)/float64(total))*100), nil
		}
	}

	return 0, fmt.Errorf("cpu line not found")
}

func memoryStats() (MemoryStats, error) {
	if runtime.GOOS != "linux" {
		return MemoryStats{}, nil
	}

	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemoryStats{}, err
	}
	var total, available uint64
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseMemValue(line)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			available = parseMemValue(line)
			break
		}
	}
	used := total - available
	return MemoryStats{
		TotalGB: round(float64(total) / 1024 / 1024),
		UsedGB:  round(float64(used) / 1024 / 1024),
		Percent: round(float64(used) / float64(total) * 100),
	}, scanner.Err()
}

func parseMemValue(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	value, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func diskStats() (DiskStats, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return DiskStats{}, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free
	return DiskStats{
		TotalGB: round(float64(total) / 1024 / 1024 / 1024),
		UsedGB:  round(float64(used) / 1024 / 1024 / 1024),
		Percent: round(float64(used) / float64(total) * 100),
	}, nil
}

func loadStats() (LoadStats, error) {
	if runtime.GOOS != "linux" {
		return LoadStats{}, nil
	}
	info := new(syscall.Sysinfo_t)
	if err := syscall.Sysinfo(info); err != nil {
		return LoadStats{}, err
	}
	return LoadStats{
		OneMinute:      round(float64(info.Loads[0]) / 65535),
		FiveMinutes:    round(float64(info.Loads[1]) / 65535),
		FifteenMinutes: round(float64(info.Loads[2]) / 65535),
	}, nil
}

func uptime() string {
	if runtime.GOOS != "linux" {
		return "n/a"
	}
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "n/a"
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "n/a"
	}
	seconds, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "n/a"
	}
	hours := int(seconds / 3600)
	return fmt.Sprintf("%dh", hours)
}

func recentLogs() []LogEntry {
	paths := []string{"/var/log/syslog", "/var/log/messages", "/var/log/nginx/error.log"}
	entries := make([]LogEntry, 0)
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			if lineNo > 20 {
				break
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			entries = append(entries, LogEntry{Source: path, Line: line, Timestamp: time.Now().UTC().Format(time.RFC3339)})
		}
		_ = file.Close()
	}
	return entries
}

func round(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func Exec(command string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("bash", "-lc", command)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
