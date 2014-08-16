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
	tmpl, _ := template.New("msg").Parse("{{.left}} {{.middle}} {{.right}}<br/>")
	line := map[string]string{"left": "", "middle": "", "right": "",}
	if isIn {
		msgs, _ := client.Cmd("ZRANGE", key(key("channel", cname), "queue"), 0, -1).List()
		for _, msg := range msgs {
			item, _ := client.Cmd("HGETALL", msg).Hash()
			msgType, _ := strconv.Atoi(item["type"])
			msgSubType, _ := strconv.Atoi(item["subtype"])
			nano, _ := strconv.ParseInt(item["time"], 10, 64)
			t := time.Unix(0, nano)
			line["left"] = t.UTC().Format(time.Stamp)

			switch msgType{
			case bot.PRIVMSG_CMD:
				if msgSubType == bot.CTCP_ACTION_SUB {
					line["middle"] = fmt.Sprintf("---ACTION:")
					line["right"] = fmt.Sprintf("%s %s", item["sender"], item["content"])
				} else {
					line["middle"] = fmt.Sprintf("<%s>",item["sender"])
					line["right"] = item["content"]
				}
			case bot.JOIN_CMD:
				line["middle"] = fmt.Sprintf("---JOIN:")
				line["right"] = fmt.Sprintf("%s JOIN %s", item["sender"], cname)
			case bot.PART_CMD:
				line["middle"] = fmt.Sprintf("---PART:")
				line["right"] = fmt.Sprintf("%s PART %s", item["sender"], cname)
			default:
				line["middle"] = fmt.Sprintf("<%s>",item["sender"])
				line["right"] = fmt.Sprintf("%s %s", bot.DMC[msgType], item["content"])
			}
			tmpl.Execute(w, line)
		}
	}
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
