package install

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/harness/godotenv/v3"
	"github.com/harness/runner/cli/install/darwin"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

type installCommand struct {
	account        string
	token          string
	url            string
	name           string
	tags           string
	configFilePath string
}

type handler interface {
	Install() error
	Start() error
	Stop() error
}

func RegisterCommands(app *kingpin.Application) {
	// register install command
	c := new(installCommand)
	cmd := app.Command("install", "Generates the config.env file for the runner.").
		Action(c.install)
	cmd.Flag("account", "account ID").
		Required().StringVar(&c.account)
	cmd.Flag("token", "runner token").
		Required().
		StringVar(&c.token)
	cmd.Flag("url", "harness server URL").
		Required().
		StringVar(&c.url)
	cmd.Flag("name", "runner name").
		Default("harnessRunner").
		StringVar(&c.name)
	cmd.Flag("tags", "runner tags").
		Default("").
		StringVar(&c.tags)
	cmd.Flag("config-file", "Path of generated config.env file.").
		Default(getDefaultConfigFilePath()).
		StringVar(&c.configFilePath)

	// register start command
	startCmd := app.Command("start", "Start runner as a service").
		Action(c.start)
	startCmd.Flag("config-file", "Path of the config.env file.").
		Default(getDefaultConfigFilePath()).
		StringVar(&c.configFilePath)

		// register stop command
	app.Command("stop", "Stop the runner service").
		Action(c.stop)

}

func (c *installCommand) install(*kingpin.ParseContext) error {
	setLogrusForCli()

	handler, err := getHandler(c.configFilePath)
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}

	configFileDir := filepath.Dir(c.configFilePath)
	// Create the directory if it doesn't exist
	err = os.MkdirAll(configFileDir, os.ModePerm)
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}

	// create the config.env file
	err = godotenv.Write(map[string]string{
		"ACCOUNT_ID": c.account,
		"TOKEN":      c.token,
		"URL":        c.url,
		"NAME":       c.name,
		"TAGS":       c.tags,
	}, c.configFilePath)
	if err != nil {
		logrus.Fatalf("Error creating config file in %s : %v", c.configFilePath, err)
	}

	err = handler.Install()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	logrus.Infof(
		"Installation completed successfully (config.env file created at %s)",
		filepath.Dir(c.configFilePath),
	)
	return nil
}

func (c *installCommand) start(*kingpin.ParseContext) error {
	setLogrusForCli()
	handler, err := getHandler(c.configFilePath)
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	err = handler.Start()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	logrus.Infof("Harness Runner started successfully")
	return nil
}

func (c *installCommand) stop(*kingpin.ParseContext) error {
	setLogrusForCli()
	handler, err := getHandler(c.configFilePath)
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	err = handler.Stop()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	logrus.Infof("Harness Runner stopped successfully")
	return nil
}

func getHandler(configFilePath string) (handler, error) {
	executablePath, err := os.Executable()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	switch operatingSystem := runtime.GOOS; operatingSystem {
	case "darwin":
		return darwin.NewDarwinHandler(executablePath, configFilePath), nil
	default:
		return nil, fmt.Errorf("harness runner cli commands not supported for Operating System: %s", operatingSystem)
	}
}

func getDefaultConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatal("Error fetching home directory: ", err)
		return ""
	}
	// Construct the path for "config.env"
	return filepath.Join(homeDir, ".harness-runner/config.env")
}

func setLogrusForCli() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})
}
