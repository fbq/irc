package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/fbq/irc/bot"
)

type LogWriter interface {
	Begin() LogWriter
	End() LogWriter
	BeginContext(name string) LogWriter
	EndContext(name string) LogWriter
	BeginItemize(name string) LogWriter
	EndItemize(name string) LogWriter
	Msg(msg *LogMsg) LogWriter
	Link(name, address string) LogWriter
	Space() LogWriter
	NewLine() LogWriter
	Write([]byte) (int, error)
}

/* HTML Log Writer */

type HtmlLogWriter struct {
	io.Writer
	location *time.Location
	tmpl     *template.Template
	//TODO add level count
}

func NewHtmlLogWriter(writer io.Writer, location *time.Location) LogWriter {
	tmpl, _ := template.New("msg").Parse("{{.left}} {{.middle}} {{.right}}")
	return &HtmlLogWriter{writer, location, tmpl}
}

func (w *HtmlLogWriter) Begin() LogWriter {
	return w.BeginContext("!doctype html").BeginContext("html").BeginContext("body")
}

func (w *HtmlLogWriter) End() LogWriter {
	return w.EndContext("body").EndContext("html")
}

func (w *HtmlLogWriter) BeginItemize(name string) LogWriter {
	return w
}

func (w *HtmlLogWriter) EndItemize(name string) LogWriter {
	return w
}

func (w *HtmlLogWriter) BeginContext(name string) LogWriter {
	fmt.Fprintf(w, "<%s>\n", name)
	return w
}

func (w *HtmlLogWriter) EndContext(name string) LogWriter {
	fmt.Fprintf(w, "</%s>\n", name)
	return w
}

func (w *HtmlLogWriter) Link(name, address string) LogWriter {
	fmt.Fprintf(w, "<a href='%s'>%s</a>", address, name)
	return w
}

func (w *HtmlLogWriter) Space() LogWriter {
	fmt.Fprintf(w, " ")
	return w
}

func (w *HtmlLogWriter) NewLine() LogWriter {
	fmt.Fprintf(w, "<br/>")
	return w
}

func (w *HtmlLogWriter) Msg(msg *LogMsg) LogWriter {

	line := map[string]string{"left": "", "middle": "", "right": ""}

	line["left"] = msg.Time.In(w.location).Format(time.Stamp)

	switch msg.Command {
	case bot.PRIVMSG_CMD:
		if msg.SubCommand == bot.CTCP_ACTION_SUB {
			line["middle"] = fmt.Sprintf("---ACTION:")
			line["right"] = fmt.Sprintf("%s %s", msg.Sender, msg.Content)
		} else {
			line["middle"] = fmt.Sprintf("<%s>", msg.Sender)
			line["right"] = msg.Content
		}
	case bot.JOIN_CMD:
		line["middle"] = fmt.Sprintf("---JOIN:")
		line["right"] = fmt.Sprintf("%s JOIN %s", msg.Sender, msg.Receiver)
	case bot.PART_CMD:
		line["middle"] = fmt.Sprintf("---PART:")
		line["right"] = fmt.Sprintf("%s PART %s", msg.Sender, msg.Receiver)
	case bot.KICK_CMD:
		line["middle"] = fmt.Sprintf("---KICK:")
		line["right"] = fmt.Sprintf("%s %s", msg.Content, msg.Info)
	default:
		line["middle"] = fmt.Sprintf("<%s>", msg.Sender)
		line["right"] = fmt.Sprintf("%s %s", bot.DMC[msg.Command], msg.Content)
	}
	w.tmpl.Execute(w, line)
	return w
}

/* Json Log Writer */

const (
	HEAD = iota
	MIDDLE
)

type JsonLogWriter struct {
	io.Writer
	encoder *json.Encoder
	state   int
}

func NewJsonLogWriter(writer io.Writer) LogWriter {
	return &JsonLogWriter{writer, json.NewEncoder(writer), 0}
}

func (w *JsonLogWriter) Begin() LogWriter {
	fmt.Fprintf(w, "{\n")
	w.state = HEAD
	return w
}

func (w *JsonLogWriter) End() LogWriter {
	fmt.Fprintf(w, "}\n")
	return w
}

func (w *JsonLogWriter) BeginContext(name string) LogWriter {
	if w.state != HEAD {
		fmt.Fprintf(w, ",")
	}
	fmt.Fprintf(w, "\"%s\" :\n", name)
	//FIXME handle nested context
	w.state = HEAD
	return w
}

func (w *JsonLogWriter) EndContext(name string) LogWriter {
	if w.state == HEAD {
		fmt.Fprintf(w, "0")
	}
	w.state = MIDDLE
	return w
}

func (w *JsonLogWriter) BeginItemize(name string) LogWriter {
	w.BeginContext(name)
	fmt.Fprintf(w, "[\n")
	return w
}

func (w *JsonLogWriter) EndItemize(name string) LogWriter {
	fmt.Fprintf(w, "]\n")
	w.state = MIDDLE
	return w
}

// Space and NewLine are meaningless for json
func (w *JsonLogWriter) Space() LogWriter {
	return w
}

func (w *JsonLogWriter) NewLine() LogWriter {
	return w
}

func (w *JsonLogWriter) Link(name string, address string) LogWriter {
	w.BeginContext(name)
	w.jsonObject(address)
	w.EndContext(name)
	return w
}

func (w *JsonLogWriter) Msg(msg *LogMsg) LogWriter {
	w.jsonObject(msg)
	return w
}

func (w *JsonLogWriter) jsonObject(o interface{}) LogWriter {
	if w.state != HEAD {
		fmt.Fprintf(w, ",")
	}
	w.encoder.Encode(o)
	w.state = MIDDLE
	return w
}
