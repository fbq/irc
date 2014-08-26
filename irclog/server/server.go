package main

import (
	"fmt"
	"strconv"
	"time"
	"net/http"
	"html/template"
	"github.com/fbq/irc/bot"
	. "github.com/fbq/irc/irclog"
	"github.com/fzzy/radix/redis"
	"github.com/drone/routes"
)

var location *time.Location
var oneDay time.Duration

const (
	PAGE_SIZE = 50
)

// poor golang, no Min for integer
func min(a, b int64) int64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func main() {
	location, _ = time.LoadLocation("Asia/Shanghai")

	oneDay, _ = time.ParseDuration("24h")

	mux := routes.New()
	mux.Get("/", index)
	mux.Get("/channel/:cname", allChannelMsg)
	mux.Get("/channel/:cname/page/:num", pagedChannelMsg)
	mux.Get("/channel/:cname/date/:year/:month/:day", datedChannelMsg)

	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

func validChannel(cname string) bool {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer client.Close()

	valid, err := client.Cmd("SISMEMBER", "channels", cname).Bool()
	if err != nil {
		return false
	}
	return valid
}

func msgCount(cname string) int64 {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer client.Close()
	// ZCARD is O(1) operation
	num, err := client.Cmd("ZCARD", Key("channel", cname, "queue")).Int64()
	if err != nil {
		fmt.Printf("%v\n", err)
		return -1
	}
	return num
}

func msgDate(cname string, index int64) time.Time {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer client.Close()

	msgs, err := client.Cmd("ZRANGE", Key("channel", cname, "queue"), index, index).List()

	if err != nil {
		fmt.Printf("%v\n", err)
		return time.Now().In(location)
	}

	nano, err := client.Cmd("HGET", msgs[0], "time").Int64()

	if err != nil {
		fmt.Printf("%v\n", err)
		return time.Now().In(location)
	}

	return time.Unix(0, nano).In(location)

}

func msgStartDate(cname string) time.Time {
	return msgDate(cname, 0)
}

func msgEndDate(cname string) time.Time {
	return msgDate(cname, -1)
}

func index(w http.ResponseWriter, r *http.Request) {
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))
	defer client.Close()

	if err != nil {
		fmt.Printf("Connection to redis server failed\n")
		return
	}

	fmt.Fprintf(w, "<!doctype html><html><body>")
	channels, _ := client.Cmd("SMEMBERS", "channels").List()

	for _, channel := range channels {
		fmt.Fprintf(w, "Channel: <a href='/channel/%s'>%s</a><br/>", channel, channel)
		count := msgCount(channel)

		fmt.Fprintf(w, "By Date:<br/>\n")

		for date := msgStartDate(channel); date.Before(msgEndDate(channel)); date = date.Add(oneDay) {
			fmt.Fprintf(w, "<a href='/channel/%s/date/%s'>%s</a> ", channel,
				date.Truncate(oneDay).Format("2006/01/02"),
				date.Truncate(oneDay).Format("2006/01/02"))
		}
		fmt.Fprintf(w, "<br/>")

		fmt.Fprintf(w, "By Page:<br/>\n")
		for i := int64(0); i < count; i += PAGE_SIZE {
			fmt.Fprintf(w, "<a href='/channel/%s/page/%v'>%v~%v</a> ", channel,
				i / PAGE_SIZE, i, min(i+PAGE_SIZE-1, count-1))
		}
		fmt.Fprintf(w, "<br/>")
		fmt.Fprintf(w, "<br/>")
	}
	fmt.Fprintf(w, "</html></body>")
}

func allChannelMsg(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	cname := params.Get(":cname")
	if (validChannel(cname)) {
		fmt.Fprintf(w, "<!doctype html><html><body>")
		channel(w, cname, 0, -1, false)
		fmt.Fprintf(w, "</html></body>")
	} else {
		fmt.Fprintf(w, `This channel is not logged now,
			if you want to add this channel in to log, 
			Ping fixme on freenode`)
	}
}

func pagedChannelMsg(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	cname := params.Get(":cname")
	num := params.Get(":num")
	pageNo, err := strconv.ParseInt(num, 10, 64)
	if err == nil && validChannel(cname) {
		fmt.Fprintf(w, "<!doctype html><html><body>")
		count := msgCount(cname)

		fmt.Fprintf(w, "<a href='/channel/%s/page/0'>First</a>", cname)
		fmt.Fprintf(w, " ")

		if pageNo > 0 {
			fmt.Fprintf(w, "<a href='/channel/%s/page/%v'>Prev</a>", cname, pageNo-1)
			fmt.Fprintf(w, " ")
		}


		if count != -1 && count >= (pageNo + 1)* PAGE_SIZE {
			fmt.Fprintf(w, "<a href='/channel/%s/page/%v'>Next</a>", cname, pageNo+1)
			fmt.Fprintf(w, " ")
		}
		fmt.Fprintf(w, "<a href='/channel/%s/page/%v'>Last</a>", cname, count / PAGE_SIZE)
		fmt.Fprintf(w, " ")

		fmt.Fprintf(w, "<a href='/channel/%s'>Full</a>", cname)
		fmt.Fprintf(w, " ")
		fmt.Fprintf(w, "<a href='/'>Home</a>")
		fmt.Fprintf(w, "<br/>\n")

		channel(w, cname, pageNo * PAGE_SIZE, (pageNo + 1) * PAGE_SIZE - 1, false)
		fmt.Fprintf(w, "</html></body>")
	} else {
		fmt.Fprintf(w, "This channel is not logged now," +
			"if you want to add this channel in to log," +
			"Ping fixme on freenode")
	}
}

func datedChannelMsg(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	cname := params.Get(":cname")
	year := params.Get(":year")
	month := params.Get(":month")
	day := params.Get(":day")

	date, err := time.ParseInLocation("2006/01/02",
		fmt.Sprintf("%s/%s/%s", year, month, day),
		location)
	if err != nil {
		fmt.Fprintf(w, "wrong date format")
	}

	start := msgStartDate(cname)
	end := msgEndDate(cname)

	fmt.Fprintf(w, "<!doctype html><html><body>")

	fmt.Fprintf(w, "<a href='/channel/%s/date/%s'>First</a>",
		cname, start.Truncate(oneDay).Format("2006/01/02"))
	fmt.Fprintf(w, " ")

	if date.Before(start) {
		fmt.Fprintf(w, "<a href='/channel/%s/date/%s'>Prev</a>",
			cname, date.AddDate(0, 0, -1).Format("2006/01/02"))
		fmt.Fprintf(w, " ")
	}

	if date.AddDate(0, 0, 1).Before(end) {
		fmt.Fprintf(w, "<a href='/channel/%s/date/%s'>Next</a>",
			cname, date.AddDate(0, 0, 1).Format("2006/01/02"))
		fmt.Fprintf(w, " ")
	}

	fmt.Fprintf(w, "<a href='/channel/%s/date/%s'>Last</a>",
		cname, end.Truncate(oneDay).Format("2006/01/02"))
	fmt.Fprintf(w, " ")

	fmt.Fprintf(w, "<a href='/channel/%s'>Full</a>", cname)
	fmt.Fprintf(w, " ")
	fmt.Fprintf(w, "<a href='/'>Home</a>")
	fmt.Fprintf(w, "<br/>\n")

	channel(w, cname, date.UnixNano(), date.AddDate(0, 0, 1).UnixNano() - 1, true)
	fmt.Fprintf(w, "</html></body>")


}
func channel(w http.ResponseWriter, cname string, start, end int64, byScore bool) {
	client, err := redis.Dial("tcp", "127.0.0.1:6379")
	defer client.Close()

	if err != nil {
		return
	}

	tmpl, _ := template.New("msg").Parse("{{.left}} {{.middle}} {{.right}}<br/>")
	line := map[string]string{"left": "", "middle": "", "right": "",}

	var msgs []string
	if byScore {
		msgs, _ = client.Cmd("ZRANGEBYSCORE", Key("channel", cname, "queue"), start, end).List()
	} else {
		msgs, _ = client.Cmd("ZRANGE", Key("channel", cname, "queue"), start, end).List()
	}
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


