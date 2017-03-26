package commands

import "github.com/urfave/cli"

var allCommands []cli.Command

func Add(c cli.Command)  {
	allCommands = append(allCommands, c)
}

func GetAll() []cli.Command {
	return allCommands
}