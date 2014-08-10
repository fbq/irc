package logbot

import (
	"net"
	"fmt"
	"bufio"
	"strings"
	"time"
	"net/http"
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
		fmt.Fprintf(conn, "join #%s\r\n", c)
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

func bot(server, nick, pass, user, info string, port uint16, channels []string,  channel chan RawMsg) {
	for { //infinite loop for reconnect
		conn, err := Connect(server, nick, pass, user, info, port, channels)
		if err == nil {
			Listen(conn, channel)
		}

	}
}

//FIXME msg in memory

var buffer []IRCMsg
var current int

func init() {
	http.HandleFunc("/", handler)
	buffer = make([]IRCMsg, 100)
	ch := make(chan RawMsg)
	go bot(SERVER, NICK, PASS, USER, INFO, PORT, CHANNELS, ch)
	go func() {
		for {
			raw := <-ch
			msg, err := ParseIRCMsg(raw.Time, raw.Line)
			if err == nil {
				buffer[current] = msg
			}
			current = (current + 1) % 100
		}
	}()
}

func handler(w http.ResponseWriter, r *http.Request) {
	for _, v := range buffer {
		fmt.Fprintf(w, "%v\n", v)
	}

}
