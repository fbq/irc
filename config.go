package logbot
const (
	SERVER = "irc.freenode.net"
	NICK = "[Olaf]"
	USER = "Olaf"
	INFO = "Olaf is a snow man, and see the log at http://xxxx" //TODO a url for the log
	PASS = ""
	PORT = uint16(6666)
)

var CHANNELS []string=[]string{"archlinux-cn", "yssyd3"} //unfortunately go dose not support const array
