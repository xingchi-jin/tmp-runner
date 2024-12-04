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
	account string
	token   string
	url     string
	name    string
	tags    string
}

type handler interface {
	Install() error
	Start() error
	Stop() error
}

func RegisterCommands(app *kingpin.Application) {
	// register install command
	c := new(installCommand)
	cmd := app.Command("install", "generates the config.env file for runner").
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

		// register start command
	app.Command("start", "start runner as a service").
		Action(start)

		// register stop command
	app.Command("stop", "stop runner service").
		Action(stop)

}

func (c *installCommand) install(*kingpin.ParseContext) error {
	setLogrusForCli()

	handler, err := getHandler()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}

	configFilePath, err := getConfigFilePath()
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
	}, configFilePath)
	if err != nil {
		logrus.Fatalf("Error creating config file in %s : %v", configFilePath, err)
	}

	err = handler.Install()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	logrus.Infof(
		"Installation completed successfully (config.env file created at %s)",
		filepath.Dir(configFilePath),
	)
	return nil
}

func start(*kingpin.ParseContext) error {
	setLogrusForCli()
	handler, err := getHandler()
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

func stop(*kingpin.ParseContext) error {
	setLogrusForCli()
	handler, err := getHandler()
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

func getHandler() (handler, error) {
	executablePath, err := os.Executable()
	if err != nil {
		logrus.Fatalf("Error: %v", err)
	}
	configFilePath, err := getConfigFilePath()
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

func getConfigFilePath() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	executableFolder := filepath.Dir(executablePath)
	// create the config.env file
	configFilePath := filepath.Join(executableFolder, "config.env")
	return configFilePath, nil
}

func setLogrusForCli() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.RFC3339,
		FullTimestamp:   true,
	})
}
