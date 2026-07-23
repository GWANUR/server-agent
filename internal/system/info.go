package system

import (
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"

	"server-agent/internal/protocol"
)

func GetServerInfo() (*protocol.ServerInfoPayload, error) {

	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	cpuName := "Unknown"

	if len(cpuInfo) > 0 {
		cpuName = cpuInfo[0].ModelName
	}

	return &protocol.ServerInfoPayload{
		Hostname: hostInfo.Hostname,
		OS:       runtime.GOOS,
		Kernel:   hostInfo.KernelVersion,
		CPU:      cpuName,
		Cores:    runtime.NumCPU(),
		RAM:      memInfo.Total,
	}, nil
}