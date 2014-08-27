package bot

import (
	"net"
	"fmt"
	"bufio"
	"strings"
	"time"
)

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


type MsgHandler func(time.Time, string, net.Conn)

func Listen(conn net.Conn, handler MsgHandler) {
	reader := bufio.NewReader(conn)

	for {
		if line, err := reader.ReadString('\n'); err == nil {
			now := time.Now()
			tokens := strings.Fields(line)
			if strings.EqualFold(tokens[0], "ping") {
				fmt.Fprintf(conn, "pong")
			}
			//fmt.Printf("%s", line)
			go handler(now, line, conn)
		} else {
			break
		}
	}
}

func Bot(config *BotConfig, handler MsgHandler) {
	for { //infinite loop for reconnect
		conn, err := Connect(config.Server, config.Nick, config.Pass, config.User, config.Info, config.Port, config.Channels)
		if err == nil {
			Listen(conn, handler)
		}

	}
}
