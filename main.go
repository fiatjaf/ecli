package main

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/fiatjaf/eclair-go"
	"github.com/go-akka/configuration"
	"github.com/itchyny/gojq"
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

	results := make([]interface{}, 5)

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		line = strings.TrimSpace(line)

		// jq filter
		if strings.HasPrefix(line, ".") {
			line = "$last | " + line
		}
		if strings.HasPrefix(line, "$") {
			line = ". as $history | $history[-2] as $prev | $history[-1] as $last | " + line

			query, err := gojq.Parse(line)
			if err != nil {
				rl.Write([]byte(err.Error() + "\n"))
				continue
			}

			iter := query.Run(results)
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}

				if err, ok := v.(error); ok {
					rl.Write([]byte(err.Error() + "\n"))
					continue
				}

				b, _ := json.MarshalIndent(v, "", "  ")
				rl.Write([]byte(string(b) + "\n"))
			}

			continue
		}

		// eclair command
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
			result := []byte(res.String())
			var val interface{}
			json.Unmarshal(result, &val)
			results = append(results[1:], val)
			b, _ := json.MarshalIndent(val, "", "  ")
			rl.Write([]byte(string(b) + "\n"))
		}
	}
}
