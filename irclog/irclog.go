package irclog
import (
	"fmt"
)

const (
	RedisServerAddress string = "127.0.0.1"
	RedisServerPort int = 6379
)

func Key(prefix, suffix string) string {
	return fmt.Sprintf("%s:%s", prefix, suffix)
}

func CountKey(prefix string) string {
	return Key(prefix, "count")
}

func RecordIdKey(prefix string, id int64) string {
	return Key(prefix, fmt.Sprintf("record:%v", id))
}
