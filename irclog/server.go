package main

import (
	"fmt"
	"strconv"
	"time"
	"net/http"
	"html/template"
	"strings"
	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
	"github.com/drone/routes"
)

const (
	redisConnStr string = "127.0.0.1:6379"
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
	[]string{"yssyd3", "archlinux-cn"},
}

var ch chan bot.RawMsg

func main() {
	ch = make(chan bot.RawMsg)
	go bot.Bot(botConfig.Server, botConfig.Nick, botConfig.Pass, botConfig.User,
		botConfig.Info, botConfig.Port, botConfig.Channels, ch)

	mux := routes.New()
	mux.Get("/", index)
	mux.Get("/channel/:cname", channel)

	http.Handle("/", mux)
	go daemon(ch)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer client.Close()

	if err != nil {
		fmt.Printf("Connection to redis server failed\n")
		return
	}

	fmt.Fprintf(w, "<!doctype html><html><body>")
	channels, _ := client.Cmd("SMEMBERS", "channels").List()

	for _, channel := range channels {
		fmt.Fprintf(w, "<a href='/channel/%s'>%s</a><br/>", channel, channel)
	}
	fmt.Fprintf(w, "</html></body>")
}

func channel(w http.ResponseWriter, r *http.Request) {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer client.Close()

	if err != nil {
		fmt.Printf("Connection to redis server failed\n")
		return
	}

	params := r.URL.Query()
	cname := params.Get(":cname")
	isIn, _ := client.Cmd("SISMEMBER", "channels", cname).Bool()
	fmt.Fprintf(w, "<!doctype html><html><body>")
	fmt.Fprintf(w, "<table>")
	fmt.Fprintf(w, "<tr><td>time</td><td>nick</td><td>type</td><td>content</td></tr>")
	tmpl, _ := template.New("msg").Parse("<tr><td>{{.time}}</td><td>{{.nick}}</td><td>{{.type}}</td><td>{{.content}}</td></tr>")
	if isIn {
		msgs, _ := client.Cmd("ZRANGE", key(cname, "queue"), 0, -1).List()
		for _, msg := range msgs {
			item, _ := client.Cmd("HGETALL", msg).Hash()
			switch item["type"] {
			case "privmsg", "PRIVMSG", "action", "ACTION":
				fmt.Fprintf(w, "<tr>")
				nano, _ := strconv.ParseInt(item["time"], 10, 64)
				t := time.Unix(0, nano)
				item["time"] = t.UTC().Format(time.UnixDate)
				tmpl.Execute(w, item)
			}
		}
	}
	fmt.Fprintf(w, "</table>")
	fmt.Fprintf(w, "</html></body>")
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


func daemon(ch chan bot.RawMsg) {
	var client *redis.Client
	client, err := redis.Dial("tcp", "127.0.0.1:6379")

	if err != nil {
		fmt.Printf("Connection to redis server failed\n")
		return
	}

	defer client.Close()
	for {
		raw := <-ch
		msg, err := bot.ParseIRCMsg(raw.Time, raw.Line)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		switch msg.Command {
		case bot.PRIVMSG_CMD:
			var prefix string
			if msg.Parameters[0][0] == '#' {
				prefix = msg.Parameters[0][1:]
			} else {
				prefix = msg.Parameters[0]
			}

			client.Cmd("SADD", "channels", prefix)
			client.Cmd("SETNX", countKey(prefix), 0)
			count := client.Cmd("INCR", countKey(prefix))

			nick := strings.Split(msg.Prefix, "!~")[0]
			id, _ := count.Int64()
			idkey := recordIdKey(prefix, id)
			queue := key(prefix, "queue")
			client.Cmd("ZADD", queue, msg.Time.UnixNano(), idkey)
			if msg.Parameters[1][0] == byte(0x01) {
				ctcpMsg := strings.Trim(msg.Parameters[1], "\x01")
				ctcpFields := strings.Split(ctcpMsg, " ")
				if strings.EqualFold(ctcpFields[0], "ACTION") {
					client.Cmd("HMSET", idkey, "time", msg.Time.UnixNano(),
						"content", ctcpFields[1], "nick", nick,
						"type", "action")
				}
			} else {
				client.Cmd("HMSET", idkey, "time", msg.Time.UnixNano(),
					"content", msg.Parameters[1], "nick", nick,
					"type", "privmsg")
			}
		}
	}
}

