package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	srv "eth-analyse-service/cmd/server"
)

type Runner interface {
	Init([]string) error
	Run() error
	Name() string
}

// ---------------- new cmd action--------------------------
func NewServerCommand() *ServerCommand {
	rs := &ServerCommand{
		fs: flag.NewFlagSet("server", flag.ExitOnError),
	}

	rs.fs.StringVar(&rs.run, "run", "required", "run applied.")

	return rs
}

type ServerCommand struct {
	fs  *flag.FlagSet
	run string
}

func (rs *ServerCommand) Name() string {
	return rs.fs.Name()
}

func (rs *ServerCommand) Init(args []string) error {
	return rs.fs.Parse(args)
}

func (rs *ServerCommand) Run() error {
	if rs.run == "required" {
		return errors.New("Provide value..")
	}
	// type action here or run with: ./cli server -h
	fmt.Println("action executed.. .", ".. with flag", rs.run)
	os.Exit(srv.RunServer())
	return nil
}

// --------------- BASE CLI-----------------------------------
func root(args []string) error {
	if len(args) < 1 {
		return errors.New("You must pass a sub-command")
	}

	cmds := []Runner{
		NewServerCommand(),
	}

	subcommand := os.Args[1]

	for _, cmd := range cmds {
		if cmd.Name() == subcommand {
			cmd.Init(os.Args[2:])
			return cmd.Run()
		}
	}

	return fmt.Errorf("Unknown subcommand: %s", subcommand)
}

func main() {
	if err := root(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
