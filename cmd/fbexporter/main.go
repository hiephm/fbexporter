package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/hiephm/fbexporter/commands"
	_ "github.com/hiephm/fbexporter/users"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Usage = "Tool for export various data from FB page"
	app.Version = "1.0.0"
	app.Commands = commands.GetAll()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config",
			Usage:  "Path to config file in JSON format, required",
			EnvVar: "FB_CONFIG_FILE",
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Error("Errors: ", err)
	}
}
