package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	RedisServerAddress string = "127.0.0.1"
	RedisServerPort    int    = 6379
)

func Key(tokens ...string) string {
	return strings.Join(tokens, ":")
}

func CountKey(prefix string) string {
	return Key(prefix, "count")
}

func RecordIdKey(prefix string, id int64) string {
	return Key(prefix, "record", strconv.FormatInt(id, 10))
}

func main() {

	if len(os.Args) == 1 { //web server is default
		server()
	} else if os.Args[1] == "server" {
		server()
	} else if os.Args[1] == "daemon" {
		if len(os.Args) > 2 {
			daemon(os.Args[2])
		} else {
			daemon("config.json")
		}
	} else {
		fmt.Printf("usage: `%s server` or `%s daemon`\n", os.Args[0], os.Args[0])
	}
}
