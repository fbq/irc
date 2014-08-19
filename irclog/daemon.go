package main

import (
	"fmt"
	"time"
	"strings"
	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
	"log"
	"os"
)

const (
	redisServerAddress string = "127.0.0.1"
	redisServerPort int = 6379
)

var location *time.Location

var ch chan bot.RawMsg

func main() {
	configFile := "config.json"
	if len(os.Args) >= 2 {
		configFile = os.Args[1]
	}

	config, err := bot.ConfigBotFromFile(configFile)

	if err != nil {
		log.Printf("Config Error: %v\n", err)
		return
	}
	ch = make(chan bot.RawMsg)
	go bot.Bot(config, ch)

	daemon(ch)
}

func key(prefix, suffix string) string {
	return fmt.Sprintf("%s:%s", prefix, suffix)
}

func countKey(prefix string) string {
	return key(prefix, "count")
}

func recordIdKey(prefix string, id int64) string {
	return key(prefix, fmt.Sprintf("record:%v", id))
}

func daemonEnd(client *redis.Client) {
	client.Close()
	log.Printf("daemon: Daemon stopped\n")
}

func daemon(ch chan bot.RawMsg) {
	var client *redis.Client
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", redisServerAddress, redisServerPort))

	if err != nil {
		log.Printf("daemon: Connection to redis server failed\n")
		return
	}

	defer daemonEnd(client)

	for {
		raw := <-ch
		msg, err := bot.ParseIRCMsg(raw.Time, raw.Line)
		if err != nil {
			fmt.Printf("loop: %v\n",  err)
			continue
		}

		sender := strings.Split(msg.Prefix, "!")[0]  //this is ok for server/user/empty

		switch msg.Command {
		case bot.PRIVMSG_CMD:
			var prefix string
			if msg.Parameters[0][0] == '#' {
				prefix = key("channel", msg.Parameters[0][1:])
				client.Cmd("SADD", "channels", msg.Parameters[0][1:])
			} else {
				prefix = key("nick", msg.Parameters[0])
			}

			id := allocMsgID(prefix, client)
			queue := key(prefix, "queue")
			client.Cmd("ZADD", queue, msg.Time.UnixNano(), id)
			client.Cmd("HMSET", id, "time", msg.Time.UnixNano(),
				"content", msg.Parameters[1], "sender", sender,
				"type", msg.Command, "subtype", msg.SubCommand)
		case bot.JOIN_CMD, bot.PART_CMD:
			prefix := key("channel", msg.Parameters[0][1:]) //only channels can be part/join
			id := allocMsgID(prefix, client)
			queue := key(prefix, "queue")
			client.Cmd("ZADD", queue, msg.Time.UnixNano(), id)
			client.Cmd("HMSET", id, "time", msg.Time.UnixNano(),
				"content", "", "sender", sender,
				"type", msg.Command, "subtype", msg.SubCommand)
		}
	}
}

func allocMsgID(prefix string, client *redis.Client) string {
	client.Cmd("SETNX", countKey(prefix), 0)
	count := client.Cmd("INCR", countKey(prefix))
	id, _ := count.Int64()
	idkey := recordIdKey(prefix, id)
	return idkey
}
