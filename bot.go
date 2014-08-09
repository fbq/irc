package main

import (
	"net"
	"fmt"
	"bufio"
	"strings"
	"time"
	"./irc"
)

type RawMsg struct {
	Time time.Time
	Line string
}

func connect(server, nick, user, info string, port uint16, channels []string) (conn net.Conn, err error){
	address := fmt.Sprintf("%s:%v", server, port)

	conn, err = net.Dial("tcp", address)

	if err != nil {
		return
	}

	fmt.Fprintf(conn, "nick %s\r\n", nick)
	fmt.Fprintf(conn, "user %s 0 * :%s\r\n", user, info)

	for _, c := range channels {
		fmt.Fprintf(conn, "join #%s\r\n", c)
	}

	return;
}


func listen(conn net.Conn, channel chan RawMsg) {
	reader := bufio.NewReader(conn)

	for {
		if line, err := reader.ReadString('\n'); err == nil {
			tokens := strings.Fields(line)
			if strings.EqualFold(tokens[0], "ping") {
				fmt.Fprintf(conn, "pong")
			}
			channel <- RawMsg{time.Now(), line}
		} else {
			break
		}
	}
}

func HourAndMinute(t time.Time) string {
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func main() {
	conn, _ := connect("irc.freenode.net", "fixme-bot", "fixme-bot", "testing", 6666, []string{"yssyd3", "archlinux-cn"})
	c := make(chan RawMsg)
	go func() {
		for {
			raw := <-c
			msg, err := irc.ParseIRCMsg(raw.Line)

			if err == nil && strings.Contains(msg.Prefix, "!"){
				fmt.Printf("%s, %s, %s, %v\n", HourAndMinute(raw.Time),
					strings.Split(msg.Prefix, "!")[0], msg.Command, msg.Paramters)
			}
		}
	}()
	listen(conn, c)
}
