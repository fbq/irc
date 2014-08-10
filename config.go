package logbot
const (
	SERVER = "<server name>"
	NICK = "<bot nick>"
	USER = "<bot username>"
	INFO = "<information>" //TODO a url for the log
	PASS = "<password, can be empty"
	PORT = uint16(6666) //port
)

var CHANNELS []string=[]string{"archlinux-cn", "yssyd3"} //unfortunately go dose not support const array
