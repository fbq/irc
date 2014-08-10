package logbot

import (
	"strings"
	"fmt"
	"time"
)

/*
 * irc message according to RFC 1459
 */
type IRCMsg struct {
	Time time.Time
	Prefix string //indicate who send out the message
	Command string
	Paramters []string
}

type InvalidIRCMsgError struct {
	Reason string
}

func (e *InvalidIRCMsgError) Error() string {
	return fmt.Sprintf("Invalid IRC Msg: %s", e.Reason)
}

func ParseIRCMsg(time time.Time, line string) (msg IRCMsg, err error) {
	line = strings.TrimSuffix(line, "\r\n")

	if len(line) == 0 {
		err = &InvalidIRCMsgError{"empty line"}
		return
	}

	msg.Time = time
	tokens := strings.Split(line, " ")

	var index int
	/* prefix */
	if strings.HasPrefix(tokens[0], ":") { // has prefix
		if len(tokens[0]) == 1 {   // space right after colon is invalid (RFC 1459 2.3)
			err = &InvalidIRCMsgError{"wrong prefix"}
			return
		}
		msg.Prefix = tokens[0][1:]
		index++
	}

	/* command */
	for ; index < len(tokens) && tokens[index] == ""; index++ {
	}

	if index == len(tokens) {
		err = &InvalidIRCMsgError{"no command"}
		return
	}

	msg.Command = tokens[index]
	index++

	/* parameter */

	/* count how many paratmeter */
	index2 := index
	count := 0
	for ; index2 <len(tokens); index2++ {
		if tokens[index2] != "" {
			count++
			if strings.HasPrefix(tokens[index2], ":") {
				break
			}
		}
	}

	msg.Paramters = make([]string, 0, count)

	for ; index < len(tokens); index++ {
		if tokens[index] != "" {
			if strings.HasPrefix(tokens[index], ":") {
				str := strings.Join(tokens[index:], " ")[1:]
				msg.Paramters = append(msg.Paramters, str)
				break
			} else {
				msg.Paramters = append(msg.Paramters, tokens[index])
			}
		}
	}

	err = nil
	return
}
