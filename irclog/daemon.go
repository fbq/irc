package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
)

func daemon(configFile string) {
	config, err := bot.ConfigBotFromFile(configFile)

	if err != nil {
		log.Printf("daemon: Config Error %v\n", err)
		return
	}
	bot.Bot(config, simpleHandler)

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

	if logMsg.Command == bot.JOIN_CMD && logMsg.Sender == "LQYMGT" {
		fmt.Fprintf(conn, "privmsg #%s :LQYMGT9", logMsg.Receiver)
	}
}
