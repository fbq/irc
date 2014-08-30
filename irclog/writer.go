package main

import (
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/fbq/irc/bot"
)

type LogWriter interface {
	Begin() LogWriter
	End() LogWriter
	Msg(msg *LogMsg) LogWriter
	Link(name, address string) LogWriter
	Space() LogWriter
	NewLine() LogWriter
}

type HtmlLogWriter struct {
	writer   io.Writer
	location *time.Location
	tmpl     *template.Template
	//TODO add level count
}

func NewHtmlLogWriter(writer io.Writer, location *time.Location) LogWriter {
	tmpl, _ := template.New("msg").Parse("{{.left}} {{.middle}} {{.right}}")
	return &HtmlLogWriter{writer, location, tmpl}
}

func (w *HtmlLogWriter) Begin() LogWriter {
	fmt.Fprintf(w.writer, "<!doctype html><html><body>")
	return w
}

func (w *HtmlLogWriter) End() LogWriter {
	fmt.Fprintf(w.writer, "</body></html>")
	return w
}

func (w *HtmlLogWriter) Link(name, address string) LogWriter {
	fmt.Fprintf(w.writer, "<a href='%s'>%s</a>", address, name)
	return w
}

func (w *HtmlLogWriter) Space() LogWriter {
	fmt.Fprintf(w.writer, " ")
	return w
}

func (w *HtmlLogWriter) NewLine() LogWriter {
	fmt.Fprintf(w.writer, "<br/>")
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
	default:
		line["middle"] = fmt.Sprintf("<%s>", msg.Sender)
		line["right"] = fmt.Sprintf("%s %s", bot.DMC[msg.Command], msg.Content)
	}
	w.tmpl.Execute(w.writer, line)
	return w
}
