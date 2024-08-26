package server

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/sirupsen/logrus"
	"os"
)

func logRunnerResourceStats() {
	processId := os.Getpid()
	logrus.Infoln("Logging resource stats")
	logrus.Infof("Runner process ID: %d", processId)

	currentProcess, err := process.NewProcess(int32(processId))
	if err != nil {
		logrus.WithError(err).Errorln("Cannot get runner process")
		return
	}

	// total CPU
	counts, err := cpu.Counts(true)
	if err != nil {
		logrus.WithError(err).Errorln("Error getting total CPU")
	} else {
		logrus.Infof("Total CPU :%d", counts)
	}

	// CPU usage of the current process
	cpuPercent, err := currentProcess.CPUPercent()
	if err != nil {
		logrus.WithError(err).Errorln("Error getting CPU usage")
	} else {
		logrus.Infof("CPU usage: %f%%", cpuPercent)
	}

	// total memory
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		logrus.WithError(err).Errorln("Error getting total memory")
	} else {
		logrus.Infof("Total memory: %vMB", virtualMemory.Total/1024/1024)
	}

	// memory usage of the current process
	memPercent, err := currentProcess.MemoryPercent()
	if err != nil {
		logrus.WithError(err).Errorln("Error getting memory usage")
	} else {
		logrus.Infof("Memory usage: %f%%", memPercent)
	}
}
