package main

import (
	"strings"

	"github.com/fiatjaf/eclair-go"
	"github.com/kballard/go-shellquote"
)

func parseCommand(line string) (string, eclair.Params) {
	line = strings.TrimSpace(line)
	argv, err := shellquote.Split(line)
	if err != nil || len(argv) == 0 {
		return "", nil
	}

	params := eclair.Params{}
	for i := 1; i < len(argv); i++ {
		arg := argv[i]

		if !strings.HasPrefix(arg, "-") {
			continue
		}

		arg = strings.TrimLeft(arg, "-")
		spl := strings.SplitN(arg, "=", 2)
		if len(spl) == 1 {
			// look at the next item in the argv array
			if len(argv) >= i+2 && !strings.HasPrefix(argv[i+1], "-") {
				// it is a valid value, so use it
				params[arg] = argv[i+1]

				// and skip it
				i++
			} else {
				// otherwise interpret it as a boolean "true" flag
				params[arg] = true
			}
		} else {
			params[spl[0]] = spl[1]
		}
	}

	return argv[0], params
}
