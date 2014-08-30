package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
)

func daemon(configFile string) {
	config, err := bot.ConfigBotFromFile(configFile)

	if err != nil {
		log.Printf("Config Error: %v\n", err)
		return
	}
	bot.Bot(config, simpleHandler)

}

func daemonHandler(t time.Time, line string, conn net.Conn) {
	var client *redis.Client
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))

	if err != nil {
		log.Printf("daemon: Connection to redis server failed\n")
		return
	}

	defer client.Close()

	msg, err := bot.ParseIRCMsg(t, line)
	if err != nil {
		log.Printf("daemon: %v\n", err)
		return
	}

	sender := strings.Split(msg.Prefix, "!")[0] //this is ok for server/user/empty

	switch msg.Command {
	case bot.PRIVMSG_CMD:
		var prefix string
		if msg.Parameters[0][0] == '#' {
			prefix = Key("channel", msg.Parameters[0][1:])
			client.Cmd("SADD", "channels", msg.Parameters[0][1:])
		} else {
			prefix = Key("nick", msg.Parameters[0])
		}

		id := allocMsgID(prefix, client)
		queue := Key(prefix, "queue")
		client.Cmd("ZADD", queue, msg.Time.UnixNano(), id)
		client.Cmd("HMSET", id, "time", msg.Time.UnixNano(),
			"content", msg.Parameters[1], "sender", sender,
			"type", msg.Command, "subtype", msg.SubCommand)
	case bot.JOIN_CMD, bot.PART_CMD:
		prefix := Key("channel", msg.Parameters[0][1:]) //only channels can be part/join
		client.Cmd("SADD", "channels", msg.Parameters[0][1:])
		id := allocMsgID(prefix, client)
		queue := Key(prefix, "queue")
		client.Cmd("ZADD", queue, msg.Time.UnixNano(), id)
		client.Cmd("HMSET", id, "time", msg.Time.UnixNano(),
			"content", "", "sender", sender,
			"type", msg.Command, "subtype", msg.SubCommand)
	}
}

func simpleHandler(t time.Time, line string, conn net.Conn) {
	var client *redis.Client
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))

	if err != nil {
		log.Printf("daemon: Connection to redis server failed\n")
		return
	}

	defer client.Close()

	msg, err := bot.ParseIRCMsg(t, line)
	if err != nil {
		log.Printf("daemon: %v\n", err)
		return
	}

	logMsg := MsgIRC2Log(&msg)

	StoreLogMsg(client, &logMsg)
}
