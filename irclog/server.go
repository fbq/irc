package main

import (
	"fmt"
	"net/http"
	"strings"
	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
)

type BotConfig struct {
	Server string
	Nick string
	Pass string
	User string
	Info string
	Port uint16
	Channels []string
}

var botConfig BotConfig = BotConfig{
	"irc.freenode.net",
	"[Olaf]",
	"xxxx",
	"bot",
	"Olaf is a snow man",
	6666,
	[]string{"yssyd3"},
}

var ch chan bot.RawMsg

func main() {
	ch = make(chan bot.RawMsg)
	go bot.Bot(botConfig.Server, botConfig.Nick, botConfig.Pass, botConfig.User,
		botConfig.Info, botConfig.Port, botConfig.Channels, ch)

	http.HandleFunc("/", index)

	go daemon(ch)
	http.ListenAndServe(":8080", nil)
}
func index(w http.ResponseWriter, r *http.Request) {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")

	if err != nil {
		fmt.Printf("Connection to redis server failed\n")
		return
	}

	fmt.Fprintf(w, "<!doctype html><html><body>")
	msgsByTime, _ := client.Cmd("zrange", "msg:bytime", 0, -1).List()

	for _, msgid := range msgsByTime {
		items, _ := client.Cmd("HGETALL", msgid).Hash()

		fmt.Fprintf(w, "%v<br/>", items)
	}
	fmt.Fprintf(w, "</html></body>")
	client.Close()
}



func daemon(ch chan bot.RawMsg) {
	var client *redis.Client
	client, err := redis.Dial("tcp", "127.0.0.1:6379")

	if err != nil {
		fmt.Printf("Connection to redis server failed\n")
		return
	}

	defer client.Close()

	client.Cmd("SETNX", "msg:count", 0)

	for {
		raw := <-ch
		msg, _ := bot.ParseIRCMsg(raw.Time, raw.Line)
		count := client.Cmd("INCR", "msg:count")
		id, _ := count.Int()
		client.Cmd("ZADD", "msg:bytime", msg.Time.UnixNano(), fmt.Sprintf("msg:id:%v", id))
		client.Cmd("HMSET", fmt.Sprintf("msg:id:%v", id),
			"prefix", msg.Prefix, "command", msg.Command,
			"parameters", strings.Join(msg.Paramters, " "),
			"time", msg.Time.UnixNano())
	}
}

