package main

import (
	"strconv"
	"strings"

	"github.com/fiatjaf/eclair-go"
)

func openFullBalance(params eclair.Params) {
	inode, ok := params["nodeId"]
	if !ok {
		printf("missing --nodeId")
		return
	}
	node := inode.(string)

	ifundingFeerateSatByte, _ := params["fundingFeerateSatByte"]
	fundingFeerateSatByte, _ := ifundingFeerateSatByte.(string)
	satPerByte, err := strconv.ParseInt(fundingFeerateSatByte, 10, 64)
	if err != nil {
		printf("missing --fundingFeerateSatByte")
		return
	}

	ln.Call("connect", eclair.Params{"nodeId": node})

	var balance int64
	if resp, err := ln.Call("onchainbalance", nil); err != nil {
		printf(": failed to call 'onchainbalance': %s", err.Error())
	} else {
		balance = resp.Get("confirmed").Int()
	}

	printf(": trying to open a channel to %s with all our balance (%d) and $satperbyte sat/b", node, balance)
	var bytes int64
	for bytes = 200; bytes < 700; bytes++ {
		fee := bytes * satPerByte
		sat := balance - fee

		printf("  : trying %d sats (fee %d sat/b)", sat, fee)

		params["fundingSatoshis"] = sat
		if _, err := ln.Call("open", params); err != nil {
			if strings.Contains(err.Error(), "Insufficient funds (code: -4)") {
				continue
			}

			printf("  : unexpected error: %s", err.Error())
			return
		}
		printf("  : success.")
	}

	printf(": stopped trying after many attempts.")
}
