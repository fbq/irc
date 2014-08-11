package logbot

import (
	"fmt"
	"net/http"
	"appengine"
	"appengine/user"
	"appengine/datastore"
//	"strings"
	"html/template"
//	"strconv"
)

func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/admin/bot", botPage)
	http.HandleFunc("/admin/bot/run", botRun)
}

func handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	q := datastore.NewQuery("msg").Ancestor(ircmsgs(c)).Order("-Time")
	count, _ := q.Count(c)
	msgs := make([]IRCMsg, 0, count)
	q.GetAll(c, &msgs)

	fmt.Fprintf(w, "<!doctype html><html><body>")
	for _, msg := range msgs {
		fmt.Fprintf(w, "%v<br/>\n", msg)
	}
	fmt.Fprintf(w, "</html></body>")
}

func botPage(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}


	tmpl, _ := template.ParseFiles("./views/bot.html")
	tmpl.Execute(w, "no data need")
}

func ircmsgs(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "msgs", "irc_msg", 0, nil)
}

func daemon(c appengine.Context, ch chan RawMsg) {
	for {
		raw := <-ch
		msg, _ := ParseIRCMsg(raw.Time, raw.Line)
		key := datastore.NewIncompleteKey(c, "msg", ircmsgs(c))
		datastore.Put(c, key, &msg)
	}
}

func botRun(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

//	server := r.PostFormValue("server")
//	nick := r.PostFormValue("nick")
//	pass := r.PostFormValue("pass")
//	user := r.PostFormValue("user")
//	info := r.PostFormValue("info")
//	uPort := r.PostFormValue("port")
//	uChannels := r.PostFormValue("channels")

//	port, _ := strconv.Atoi(uPort)
//
//	channels := strings.Split(uChannels, " ")

	ch := make(chan RawMsg)
	go bot(SERVER, NICK, PASS, USER, INFO, uint16(PORT), CHANNELS, ch)
	go daemon(c, ch)
}


