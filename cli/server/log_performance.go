package server

import (
	"context"
	"os"

	"github.com/harness/runner/logger"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

func logRunnerResourceStats(ctx context.Context) {
	processId := os.Getpid()
	logger.Infoln(ctx, "Logging resource stats")
	logger.Infof(ctx, "Runner process ID: %d", processId)

	currentProcess, err := process.NewProcess(int32(processId))
	if err != nil {
		logger.WithError(ctx, err).Errorln("Cannot get runner process")
		return
	}

	// total CPU
	counts, err := cpu.Counts(true)
	if err != nil {
		logger.WithError(ctx, err).Errorln("Error getting total CPU")
	} else {
		logger.Infof(ctx, "Total CPU :%d", counts)
	}

	// CPU usage of the current process
	cpuPercent, err := currentProcess.CPUPercent()
	if err != nil {
		logger.WithError(ctx, err).Errorln("Error getting CPU usage")
	} else {
		logger.Infof(ctx, "CPU usage: %f%%", cpuPercent)
	}

	// total memory
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		logger.WithError(ctx, err).Errorln("Error getting total memory")
	} else {
		logger.Infof(ctx, "Total memory: %vMB", virtualMemory.Total/1024/1024)
	}

	// memory usage of the current process
	memPercent, err := currentProcess.MemoryPercent()
	if err != nil {
		logger.WithError(ctx, err).Errorln("Error getting memory usage")
	} else {
		logger.Infof(ctx, "Memory usage: %f%%", memPercent)
	}
}
