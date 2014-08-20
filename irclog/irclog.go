package irclog
import (
	"strings"
	"strconv"
)

const (
	RedisServerAddress string = "127.0.0.1"
	RedisServerPort int = 6379
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
