package logbot

import (
	"net"
	"fmt"
	"bufio"
	"strings"
	"time"
)

type RawMsg struct {
	Time time.Time
	Line string
}

func Connect(server, nick, pass, user, info string, port uint16, channels []string) (conn net.Conn, err error){
	address := fmt.Sprintf("%s:%v", server, port)

	conn, err = net.Dial("tcp", address)

	if err != nil {
		return
	}

	fmt.Fprintf(conn, "nick %s\r\n", nick)
	fmt.Fprintf(conn, "user %s 0 * :%s\r\n", user, info)

	if pass != "" {
		fmt.Fprintf(conn, "privmsg NickServ :identify %s\r\n", pass)
	}

	for _, c := range channels {
		if c != "" {
			fmt.Fprintf(conn, "join #%s\r\n", c)
		}
	}

	return;
}


func Listen(conn net.Conn, channel chan RawMsg) {
	reader := bufio.NewReader(conn)

	for {
		if line, err := reader.ReadString('\n'); err == nil {
			tokens := strings.Fields(line)
			if strings.EqualFold(tokens[0], "ping") {
				fmt.Fprintf(conn, "pong")
			}
			//fmt.Printf("%s", line)
			channel <- RawMsg{time.Now(), line}
		} else {
			break
		}
	}
}

func Bot(server, nick, pass, user, info string, port uint16, channels []string,  channel chan RawMsg) {
	for { //infinite loop for reconnect
		conn, err := Connect(server, nick, pass, user, info, port, channels)
		if err == nil {
			Listen(conn, channel)
		}

	}
}
