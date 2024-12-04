package darwin

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
)

const SVC_NAME = "harness.runner"

//go:embed runner.plist.template
var PLIST_TEMPLATE string

type ServiceConfig struct {
	SvcName    string
	RunnerRoot string
	RunnerPath string
	ConfigPath string
	UserName   string
	StdoutPath string
	StderrPath string
}

type DarwinHandler struct {
	executablePath string
	configFilePath string
}

func NewDarwinHandler(executablePath, configFilePath string) *DarwinHandler {
	return &DarwinHandler{executablePath: executablePath, configFilePath: configFilePath}
}

func (d *DarwinHandler) Install() error {
	// Nothing to do here, as the parent
	// install handler (caller of this function)
	// already creates the config.env file.
	return nil
}

func (d *DarwinHandler) Start() error {
	err := checkConfigFileExists(d.configFilePath)
	if err != nil {
		return err
	}
	// Check if the service is running
	isRunning, err := isServiceRunning()
	if err != nil {
		return fmt.Errorf("Error checking service status: %v", err)
	}
	if isRunning {
		return fmt.Errorf("Runner service %s is currently running. To restart it,"+
			"use `./harness-runner stop` and then `./harness-runner start`", SVC_NAME)
	}

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("Error getting current user: %v", err)
	}

	svcConfig := &ServiceConfig{
		SvcName:    SVC_NAME,
		RunnerRoot: filepath.Dir(d.executablePath),
		RunnerPath: d.executablePath,
		ConfigPath: d.configFilePath,
		UserName:   currentUser.Name,
		StdoutPath: getStdoutPath(),
		StderrPath: getStderrPath(),
	}

	// Create the plist file and load the service
	logrus.Infof("Setting up the service...")
	plistPath := getPlistPath()
	err = createPlist(svcConfig, plistPath)
	if err != nil {
		return fmt.Errorf("Error creating plist file: %v", err)
	}

	// Create the plist file and load the service
	logrus.Infof("Starting up the service...")
	err = loadService(plistPath)
	if err != nil {
		return fmt.Errorf("Error loading service: %v", err)
	}
	logrus.Infof("Logs being stored in: %s", getStderrPath())
	return nil
}

func (d *DarwinHandler) Stop() error {
	// Check if the service is running
	isRunning, err := isServiceRunning()
	if err != nil {
		return fmt.Errorf("Error checking service status: %v", err)
	}
	if !isRunning {
		return fmt.Errorf("service %s is not currently running", SVC_NAME)
	}
	logrus.Infof("Stopping the service %s...", SVC_NAME)

	err = unloadService()
	if err != nil {
		return fmt.Errorf("Error unloading service: %v", err)
	}

	// Poll until the service is no longer running
	for {
		isRunning, err = isServiceRunning()
		if err != nil {
			return fmt.Errorf("Error checking service status: %v", err)
		}
		if !isRunning {
			break
		}
		logrus.Infof("Waiting for service %s to stop...", SVC_NAME)
		time.Sleep(3 * time.Second)
	}

	return nil
}

// Create the plist file for the runner service
func createPlist(config *ServiceConfig, plistPath string) error {
	// Ensure plist directory exists
	plistDir := filepath.Dir(plistPath)
	if _, err := os.Stat(plistDir); os.IsNotExist(err) {
		err = os.MkdirAll(plistDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create folder %s : %v", plistDir, err)
		}
	}

	// Parse and execute the embedded plist template
	tmpl, err := template.New("plist").Parse(PLIST_TEMPLATE)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %v", err)
	}

	plistFile, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %v", err)
	}
	defer plistFile.Close()

	err = tmpl.Execute(plistFile, config)
	if err != nil {
		return fmt.Errorf("failed to write plist template: %v", err)
	}

	logrus.Infof("Created .plist file at %s", plistPath)
	return nil
}

func checkConfigFileExists(configFilePath string) error {
	_, err := os.Stat(configFilePath)
	if err != nil {
		return fmt.Errorf("no config file found at %s", configFilePath)
	}
	return nil
}

// Check if a service is running using launchctl
func isServiceRunning() (bool, error) {
	cmd := exec.Command("launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), SVC_NAME), nil
}

// Remove a service (and stop it) using launchctl
func unloadService() error {
	cmd := exec.Command("launchctl", "remove", SVC_NAME)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unload service: %v", err)
	}
	return nil
}

// Load a service (and start it) using launchctl
func loadService(plistPath string) error {
	cmd := exec.Command("launchctl", "load", "-w", plistPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load service: %v", err)
	}
	logrus.Infof("Service started from %s", plistPath)
	return nil
}

func getStdoutPath() string {
	return filepath.Join(getLibraryPath(), "Logs", SVC_NAME, "stdout.log")
}

func getStderrPath() string {
	return filepath.Join(getLibraryPath(), "Logs", SVC_NAME, "stderr.log")
}

func getPlistPath() string {
	fileName := SVC_NAME + ".plist"
	if os.Geteuid() == 0 {
		// Running as root
		return filepath.Join(getLibraryPath(), "/LaunchDaemons", fileName)
	}
	return filepath.Join(getLibraryPath(), "/LaunchAgents", fileName)
}

func getLibraryPath() string {
	// Running as root
	if os.Geteuid() == 0 {
		return "/Library"
	}
	return filepath.Join(os.Getenv("HOME"), "/Library")
}
