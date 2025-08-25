package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/retawsolit/WeMeet-recorder/helpers"
	"github.com/retawsolit/WeMeet-recorder/pkg/config"
	"github.com/retawsolit/WeMeet-recorder/pkg/controllers"
	"github.com/retawsolit/WeMeet-recorder/version"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func main() {
	cli.VersionPrinter = func(c *cli.Command) {
		fmt.Printf("%s\n", c.Version)
	}

	app := &cli.Command{
		Name:        "WeMeet-recorder",
		Usage:       "Recording system for WeMeet",
		Description: "without option will start server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Usage:       "Configuration file",
				DefaultText: "config.yaml",
				Value:       "config.yaml",
			},
		},
		Action:  startServer,
		Version: version.Version,
	}
	err := app.Run(context.Background(), os.Args)
	if err != nil {
		logrus.Fatalln(err)
	}
}

func startServer(ctx context.Context, c *cli.Command) error {
	appCnf, err := helpers.ReadYamlConfigFile(c.String("config"))
	if err != nil {
		logrus.Fatalln(err)
	}
	// set this config for global usage
	config.New(appCnf)

	// now prepare our server
	err = helpers.PrepareServer(config.GetConfig())
	if err != nil {
		logrus.Fatalln(err)
	}

	// start services
	rc := controllers.NewRecorderController()
	go rc.BootUp()

	// defer close connections
	defer helpers.HandleCloseConnections()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logrus.Infoln("exit requested, shutting down signal", sig)
	// close all the remaining task
	rc.CallEndToAll()

	return nil
}
