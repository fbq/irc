package logbot

import (
	"time"
	"strings"
	"fmt"
	"testing"
)

func HourAndMinute(t time.Time) string {
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func TestBot(t *testing.T) {
	c := make(chan RawMsg)

	go bot(SERVER, NICK, PASS, USER, INFO, PORT, CHANNELS, c)
	for {
		raw := <-c
		msg, err := ParseIRCMsg(raw.Time, raw.Line)

		if err == nil && strings.Contains(msg.Prefix, "!") {
			fmt.Printf("%s, %s, %s, %v\n", HourAndMinute(msg.Time),
				strings.Split(msg.Prefix, "!")[0], msg.Command, msg.Paramters)
		}
	}
}
