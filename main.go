package main

import (
	"path/filepath"

	"github.com/chzyer/readline"
	"github.com/fiatjaf/eclair-go"
	"github.com/go-akka/configuration"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
)

func main() {
	var dir string
	pflag.StringVar(&dir, "dir", "~/.eclair", "Directory where eclair.conf is.")
	dir, _ = homedir.Expand(dir)

	conf := filepath.Join(dir, "eclair.conf")
	p := configuration.LoadConfig(conf)
	alias := p.GetString("eclair.node-alias", "eclair")
	host := p.GetString("eclair.api.binding-ip", "127.0.0.1")
	port := p.GetString("eclair.api.port", "8080")
	password := p.GetString("eclair.api.password", "satoshi21")

	readline.SetHistoryPath(filepath.Join(dir, "ecli.history"))

	ln := eclair.Client{Host: "http://" + host + ":" + port, Password: password}

	rl, err := readline.New(alias + "> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	if ws, err := ln.Websocket(); err != nil {
		panic(err)
	} else {
		go func() {
			for message := range ws {
				typ := message.Get("type").String()
				if typ == "channel-state-changed" {
					continue
				}

				rl.Write([]byte("[" + typ + "] " + message.String() + "\n"))
			}
		}()
	}

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		command, params := parseCommand(line)
		if command == "" || params == nil {
			continue
		}

		res, err := ln.Call(command, params)
		if err != nil {
			if err.Error() == "eclair said: Substream Source(EntitySource) cannot be materialized more than once" {
				continue
			}

			rl.Write([]byte(err.Error() + "\n"))
		} else {
			rl.Write([]byte(res.String() + "\n"))
		}
	}
}
