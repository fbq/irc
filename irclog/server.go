package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/drone/routes"
	"github.com/fzzy/radix/redis"
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

func server() {
	location, _ = time.LoadLocation("Asia/Shanghai")

	oneDay, _ = time.ParseDuration("24h")

	mux := routes.New()
	mux.Get("/", index)
	mux.Get(":format(/json)?/channel/:cname", allChannelMsg)
	mux.Get(":format(/json)?/channel/:cname/page/:num", pagedChannelMsg)
	mux.Get(":format(/json)?/channel/:cname/date/:year/:month/:day", datedChannelMsg)

	http.Handle("/", mux)
	http.ListenAndServe(":8080", nil)
}

func validChannel(cname string) bool {
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))
	defer client.Close()

	valid, err := client.Cmd("SISMEMBER", "channels", cname).Bool()
	if err != nil {
		return false
	}
	return valid
}

func msgCount(cname string) int64 {
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))
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
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))
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

		for date := TruncateInLocation(msgStartDate(channel), oneDay); !date.After(msgEndDate(channel)); date = date.Add(oneDay) {
			fmt.Fprintf(w, "<a href='/channel/%s/date/%s'>%s</a> ", channel,
				date.Format("2006/01/02"),
				date.Format("2006/01/02"))
		}
		fmt.Fprintf(w, "<br/>")

		fmt.Fprintf(w, "By Page:<br/>\n")
		for i := int64(0); i < count; i += PAGE_SIZE {
			fmt.Fprintf(w, "<a href='/channel/%s/page/%v'>%v~%v</a> ", channel,
				i/PAGE_SIZE, i, min(i+PAGE_SIZE-1, count-1))
		}
		fmt.Fprintf(w, "<br/>")
		fmt.Fprintf(w, "<br/>")
	}
	fmt.Fprintf(w, "</html></body>")
}

func allChannelMsg(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	cname := params.Get(":cname")
	if validChannel(cname) {
		format := params.Get(":format")

		var writer LogWriter
		if format == "/json" {
			writer = NewJsonLogWriter(w)
		} else {
			writer = NewHtmlLogWriter(w, location)
		}
		WriteAllMsgInChannel(writer, cname)
	} else {
		http.NotFound(w, r)
	}
}

func pagedChannelMsg(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	cname := params.Get(":cname")
	num := params.Get(":num")
	pageNo, err := strconv.ParseInt(num, 10, 64)

	if err == nil && validChannel(cname) {
		format := params.Get(":format")
		var writer LogWriter
		if format == "/json" {
			writer = NewJsonLogWriter(w)
		} else {
			writer = NewHtmlLogWriter(w, location)
		}

		WriteMsgInChannelByPage(writer, cname, pageNo)
	} else {
		http.NotFound(w, r)
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
	if err == nil && validChannel(cname) {
		format := params.Get(":format")

		var writer LogWriter
		if format == "/json" {
			writer = NewJsonLogWriter(w)
		} else {
			writer = NewHtmlLogWriter(w, location)
		}
		WriteMsgInChannelByDate(writer, cname, date)
	} else {
		http.NotFound(w, r)
	}

}

func WriteAllMsgInChannel(writer LogWriter, cname string) {
	writer.Begin()
	writer.Link("Json", fmt.Sprintf("/json/channel/%s", cname))
	writer.NewLine()
	channel(writer, cname, 0, -1, false)
	writer.End()
}

func WriteMsgInChannelByPage(writer LogWriter, cname string, pageNo int64) {
	count := msgCount(cname)

	writer.Begin()
	writer.Link("First", fmt.Sprintf("/channel/%s/page/0", cname))
	writer.Space()

	if pageNo > 0 {
		writer.Link("Prev", fmt.Sprintf("/channel/%s/page/%v", cname, pageNo-1))
		writer.Space()
	}

	if count != -1 && count >= (pageNo+1)*PAGE_SIZE {
		writer.Link("Next", fmt.Sprintf("/channel/%s/page/%v", cname, pageNo+1))
		writer.Space()
	}
	writer.Link("Last", fmt.Sprintf("/channel/%s/page/%v", cname, count/PAGE_SIZE))
	writer.Space()

	writer.Link("Full", fmt.Sprintf("/channel/%s", cname))
	writer.Space()
	writer.Link("Home", "/")
	writer.Space()
	writer.Link("Json", fmt.Sprintf("/json/channel/%s/page/%v", cname, pageNo))
	writer.NewLine()

	channel(writer, cname, pageNo*PAGE_SIZE, (pageNo+1)*PAGE_SIZE-1, false)
	writer.End()
}

func WriteMsgInChannelByDate(writer LogWriter, cname string, date time.Time) {
	start := msgStartDate(cname)
	end := msgEndDate(cname)

	writer.Begin()

	writer.Link("First",
		fmt.Sprintf("/channel/%s/date/%s",
			cname, TruncateInLocation(start, oneDay).Format("2006/01/02")))
	writer.Space()

	if date.After(start) {
		writer.Link("Prev",
			fmt.Sprintf("/channel/%s/date/%s",
				cname, date.AddDate(0, 0, -1).Format("2006/01/02")))
		writer.Space()
	}

	if date.AddDate(0, 0, 1).Before(end) {
		writer.Link("Next",
			fmt.Sprintf("/channel/%s/date/%s",
				cname, date.AddDate(0, 0, 1).Format("2006/01/02")))
		writer.Space()
	}

	writer.Link("Last",
		fmt.Sprintf("/channel/%s/date/%s",
			cname, TruncateInLocation(end, oneDay).Format("2006/01/02")))
	writer.Space()

	writer.Link("Full", fmt.Sprintf("/channel/%s", cname))
	writer.Space()
	writer.Link("Home", "/")
	writer.Space()
	writer.Link("Json", fmt.Sprintf("/json/channel/%s/date/%s", cname, date.Format("2006/01/02")))
	writer.NewLine()

	channel(writer, cname, date.UnixNano(), date.AddDate(0, 0, 1).UnixNano()-1, true)
	writer.End()
}

func channel(writer LogWriter, cname string, start, end int64, byScore bool) {
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%v", RedisServerAddress, RedisServerPort))
	defer client.Close()

	if err != nil {
		return
	}

	var msgs []string
	if byScore {
		msgs, _ = client.Cmd("ZRANGEBYSCORE", Key("channel", cname, "queue"), start, end).List()
	} else {
		msgs, _ = client.Cmd("ZRANGE", Key("channel", cname, "queue"), start, end).List()
	}

	writer.BeginItemize("msgs")
	for _, msg := range msgs {
		item, _ := client.Cmd("HGETALL", msg).Hash()
		msgType, _ := strconv.Atoi(item["type"])
		msgSubType, _ := strconv.Atoi(item["subtype"])
		nano, _ := strconv.ParseInt(item["time"], 10, 64)
		m := LogMsg{Time: time.Unix(0, nano),
			Command:    msgType,
			SubCommand: msgSubType,
			Content:    item["content"],
			Info:       item["info"],
			Sender:     item["sender"],
			Receiver:   cname}

		writer.Msg(&m).NewLine()
	}
	writer.EndItemize("msgs")
}
