package logbot

import (
	"time"
	"strings"
	"fmt"
	"testing"
)


const (
	server = "irc.freenode.net"
	nick = "[Olaf]"
	user = "Olaf"
	info = "Olaf is a snow man, and see the log at http://xxxx" //TODO a url for the log
	pass = ""
	port = uint16(6666)
)

var channels []string=[]string{"archlinux-cn", "yssyd3"} //unfortunately go dose not support const array

func hourAndMinute(t time.Time) string {
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func TestBot(t *testing.T) {
	c := make(chan RawMsg)

	go Bot(server, nick, pass, user, info, port, channels, c)
	for {
		raw := <-c
		msg, err := ParseIRCMsg(raw.Time, raw.Line)

		if err == nil && strings.Contains(msg.Prefix, "!") {
			fmt.Printf("%s, %s, %s, %v\n", hourAndMinute(msg.Time),
				strings.Split(msg.Prefix, "!")[0], msg.Command, msg.Paramters)
		}
	}
}
