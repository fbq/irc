package bot

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
	Command int
	SubCommand int
	Parameters []string
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

	msg.Command = CMD[strings.ToUpper(tokens[index])]
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

	msg.Parameters = make([]string, 0, count)

	for ; index < len(tokens); index++ {
		if tokens[index] != "" {
			if strings.HasPrefix(tokens[index], ":") {
				str := strings.Join(tokens[index:], " ")[1:]
				msg.Parameters = append(msg.Parameters, str)
				break
			} else {
				msg.Parameters = append(msg.Parameters, tokens[index])
			}
		}
	}

	if msg.Command == PRIVMSG_CMD { // could be a CTCP
		if len(msg.Parameters[1]) > 2 && msg.Parameters[1][0] == '\x01' {
			str := strings.Trim(msg.Parameters[1], "\x01")
			if i := strings.Index(str, " "); i != -1 {
				if strings.EqualFold("ACTION", str[0:i]) {
					msg.SubCommand = CTCP_ACTION_SUB
					msg.Parameters[1] = str[i+1:]
				}
			}
		}
	}
	err = nil
	return
}
