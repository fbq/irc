package bot

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

var channels []string = []string{"archlinux-cn", "yssyd3"} //unfortunately go dose not support const array

func hourAndMinute(t time.Time) string {
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func TestBot(t *testing.T) {
	config, err := ConfigBotFromFile("config.json")
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	Bot(config, func(t time.Time, line string, conn net.Conn) {
		msg, err := ParseIRCMsg(t, line)
		if err == nil && strings.Contains(msg.Prefix, "!") {
			fmt.Printf("%s, %s, %s, %v, %v\n", hourAndMinute(msg.Time),
				strings.Split(msg.Prefix, "!")[0], DMC[msg.Command], msg.SubCommand, msg.Parameters)
		}
	})
}
