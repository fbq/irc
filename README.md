irc
======

irc related projects written by golang

STATUS: under developing

* `github.com/fbq/irc/bot` is a library for developing an irc bot
* `github.com/fbq/irc/irclog` is a web server for log in irc channels
    * USEAGE
        * Run a irc bot to collect irc messages, run `irclog daemon <config file>`,
          the config file is a json-format config for irc bot
        * Run a web server to show the log, run `irclog server`, however `irclog` is also default for runing server
    * LogWriter
        * Log writer is an interface that abstract the output structure of irc log msg
	* To output the log msgs in the different format, just implement another LogWriter

    * TODO
        * <del>join/quit and other msg support</del>
        * <del>a better data structure for log records</del>
        * err handling
        * search
        * chat via web
