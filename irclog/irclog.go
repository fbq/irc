package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
)

const (
	RedisServerAddress string = "127.0.0.1"
	RedisServerPort    int    = 6379
)

type LogMsg struct {
	Sender     string
	Receiver   string
	Content    string
	Command    int
	SubCommand int
	Time       time.Time
	ToUser     bool `json:"-"`
}

// Poor golang, can not truncate in other zone
func TruncateInLocation(t time.Time, d time.Duration) time.Time {
	_, offset := t.Zone()
	if offset == 0 {
		return t.Truncate(d)
	} else {
		duration := time.Duration(offset) * time.Second
		return t.Add(duration).Truncate(d).Add(-duration)
	}
}

// one-way convert function from a IRC msg to a log msg
func MsgIRC2Log(msg *bot.IRCMsg) (logMsg LogMsg) {
	logMsg.Sender = strings.Split(msg.Prefix, "!")[0] //this is ok for server/user/empty
	logMsg.Time = msg.Time
	logMsg.Command = msg.Command
	logMsg.SubCommand = msg.SubCommand

	switch msg.Command {
	case bot.PRIVMSG_CMD:
		if msg.Parameters[0][0] == '#' {
			logMsg.Receiver = msg.Parameters[0][1:]
		} else {
			logMsg.Receiver = msg.Parameters[0]
			logMsg.ToUser = true
		}

		logMsg.Content = msg.Parameters[1]

	case bot.JOIN_CMD, bot.PART_CMD:
		logMsg.Receiver = msg.Parameters[0][1:] //only channels can be part/join

	case bot.KICK_CMD:
		logMsg.Receiver = msg.Parameters[0][1:] //only channels

		logMsg.Content = fmt.Sprintf("Kick out %s for %s", msg.Parameters[1], msg.Parameters[2])

	}
	return
}

func allocMsgID(prefix string, client *redis.Client) string {
	client.Cmd("SETNX", CountKey(prefix), 0)
	count := client.Cmd("INCR", CountKey(prefix))
	id, _ := count.Int64()
	idkey := RecordIdKey(prefix, id)
	return idkey
}

func allocMsgIDandStore(prefix string, msg *LogMsg, client *redis.Client) {
	id := allocMsgID(prefix, client)
	queue := Key(prefix, "queue")
	client.Cmd("ZADD", queue, msg.Time.UnixNano(), id)
	client.Cmd("HMSET", id, "time", msg.Time.UnixNano(),
		"content", msg.Content, "sender", msg.Sender,
		"type", msg.Command, "subtype", msg.SubCommand)
}
func StoreLogMsg(client *redis.Client, msg *LogMsg) {
	var prefix string
	switch msg.Command {
	case bot.PRIVMSG_CMD:
		if msg.ToUser {
			prefix = Key("nick", msg.Receiver)
		} else {
			prefix = Key("channel", msg.Receiver)
		}

	case bot.JOIN_CMD, bot.PART_CMD:
		prefix = Key("channel", msg.Receiver) //only channels can be part/join

	case bot.KICK_CMD:
		prefix = Key("channel", msg.Receiver) //only channels

	default:
		return //short cut
	}

	if !msg.ToUser {
		client.Cmd("SADD", "channels", msg.Receiver)
	}

	allocMsgIDandStore(prefix, msg, client)
}

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
