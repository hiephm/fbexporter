package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/hiephm/fbexporter/commands"
	_ "github.com/hiephm/fbexporter/users"
	"github.com/urfave/cli"
)

func main() {
	initLogger(log.StandardLogger(), "info")
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

func initLogger(logger *log.Logger, logLevel string) {
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	logger.Formatter = formatter
	switch logLevel {
	case "debug":
		logger.Level = log.DebugLevel
	case "error":
		logger.Level = log.ErrorLevel
	default:
		logger.Level = log.InfoLevel
	}
}
