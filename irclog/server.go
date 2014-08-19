package main

import (
	"fmt"
	"strconv"
	"time"
	"net/http"
	"html/template"
	"github.com/fbq/irc/bot"
	"github.com/fzzy/radix/redis"
	"github.com/drone/routes"
)

const (
	redisServerAddress string = "127.0.0.1"
	redisServerPort int = 6379
)

var location *time.Location


var ch chan bot.RawMsg

func main() {
	location, _ = time.LoadLocation("Asia/Shanghai")

	mux := routes.New()
	mux.Get("/", index)
	mux.Get("/channel/:cname", channel)

	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", redisServerAddress, redisServerPort))
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
			line["left"] = t.In(location).Format(time.Stamp)

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
