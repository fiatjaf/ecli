package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/fiatjaf/eclair-go"
	"github.com/go-akka/configuration"
	"github.com/itchyny/gojq"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"
)

var ln eclair.Client
var rl *readline.Instance

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

	ln = eclair.Client{Host: "http://" + host + ":" + port, Password: password}

	if rl_, err := readline.New(alias + "> "); err != nil {
		panic(err)
	} else {
		rl = rl_
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

				printf("[%s] %s", typ, message.String())
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
	jqfilter:
		if strings.HasPrefix(line, ".") {
			line = "$last | " + line
		}
		if strings.HasPrefix(line, "$") {
			line = ". as $history | $history[-2] as $prev | $history[-1] as $last | " + line

			query, err := gojq.Parse(line)
			if err != nil {
				printf("<> %s", err.Error())
				continue
			}

			iter := query.Run(results)
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}

				if err, ok := v.(error); ok {
					printf("<> %s", err.Error())
					continue
				}

				b, _ := json.MarshalIndent(v, "", "  ")
				printf(string(b))
			}

			continue
		}

		// command + jq filter
		var jqFilterAtTheEnd string
		if spl := strings.SplitN(line, "|", 2); len(spl) == 2 {
			jqFilterAtTheEnd = strings.TrimSpace(spl[1])
			line = strings.TrimSpace(spl[0])
		}

		// command
		command, params := parseCommand(line)
		if command == "" || params == nil {
			continue
		}
		var (
			res    gjson.Result
			cmdErr error
		)

		/// extra command
		switch command {
		case "openfullbalance":
			res, cmdErr = openFullBalance(params)
		default:
			/// eclair command
			res, cmdErr = ln.Call(command, params)
		}

		if cmdErr != nil {
			printf("<> %s", cmdErr.Error())
		} else {
			result := []byte(res.String())
			var val interface{}
			json.Unmarshal(result, &val)
			results = append(results[1:], val)

			if jqFilterAtTheEnd != "" {
				line = jqFilterAtTheEnd
				goto jqfilter
			} else {
				b, _ := json.MarshalIndent(val, "", "  ")
				printf(string(b))
				continue
			}
		}
	}
}

func printf(base string, args ...interface{}) {
	rl.Write([]byte(fmt.Sprintf(base, args...) + "\n"))
}
