// This package contains an issue managing slackbot that
// listens to slash commands on HTTP ports, invokes github
// queryies in response and reports on the results.
package slack

import (
	"fmt"
	"net/http"
	"strings"
	"strconv"
	"sync"

	"github.com/ctelfer-docker/slkiss/github"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithFields(logrus.Fields{"component": "slackbot"})

type botHandlerFunc func(*IssueBot, http.ResponseWriter, *http.Request, []string)

var handlers = map[string]botHandlerFunc{
	"help":       help,
	"find":       findIssue,
	"close":      closeIssue,
	"reopen":     reopenIssue,
	"assign":     assignIssue,
	"unassign":   unassignIssue,
	"register":   registerUser,
	"get-alias":  getAlias,
	"unregister": unregisterUser,
}

// Bot implements a slackbot that manages 
type IssueBot struct {
	sync.Mutex
	addr     string
	mux      *http.ServeMux
	agent    *github.Agent
	dispatch map[string]botHandlerFunc
	g2s      map[string]string
	s2g      map[string]string
}


// This structure basically hides the ServeHTTP method
// while providing access to the IssueBot structure for serving.
type botHandlerCtx struct {
	b *IssueBot
}

// Create a new IssueBot
func NewIssueBot(addr string, repo string) *IssueBot {
	b := &IssueBot{}
	b.addr = addr
	b.mux = http.NewServeMux()
	b.agent = github.NewRepoAgent(repo)
	b.mux.Handle("/issue", &botHandlerCtx{b})
	b.g2s = make(map[string]string)
	b.s2g = make(map[string]string)
	return b
}

// Set the authentication token to send with Github requests
func (b *IssueBot) SetGithubAuth(token string) {
	b.agent.SetToken(token)
}

// Add a mapping from a slack username (sname) to a github username (gname).
func (b *IssueBot) AddUserMap(sname string, gname string) bool{
	if _, ok := b.g2s[gname]; ok {
		return false
	}
	if _, ok := b.s2g[sname]; ok {
		return false
	}
	b.g2s[gname] = sname
	b.s2g[sname] = gname
	return true
}

// Delete the name mappings for the user specified by the slack name sname.
func (b *IssueBot) DelUserBySlack(sname string) {
	gname, ok := b.s2g[sname]
	if ok {
		delete(b.g2s, gname)
		delete(b.s2g, sname)
	}

}

// Start the http server in the issuebot
func (b *IssueBot) Run() {
	log.Fatal(http.ListenAndServe(b.addr, b.mux))
}

// This is the main dispatch for the /issue command from slack.
// It performs the basic command parsing and then calls a handler for
// the subcommand.
func (c *botHandlerCtx)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := c.b
	if err := r.ParseForm(); err != nil {
		reqErr(log, w, err)
		return
	}
	text, err := getField("text", r)
	if err != nil {
		reqErr(log, w, err)
		return
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		help(b, w, r, nil)
		return
	}

	h, ok := handlers[fields[0]]
	if !ok {
		h = help
	}
	h(b, w, r, fields[1:])
}

func help(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	w.Write([]byte(`usage: /issue CMD [params]
Commands:
	/issue find NUM
	/issue close NUM
	/issue reopen NUM
	/issue assign NUM [@SLACKNAME|@me|GITHUBNAME]
	/issue unassign NUM
	/issue register GITHUBUSER
	/issue get-alias
	/issue unregister
`))
}

func findIssue(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	var assignee string

	log := log.WithField("method", "findIssue")
	msg := "usage: /issue find NUMBER"
	defer func(){w.Write([]byte(msg))}()

	inum, ok := parseSimpleNumCmd(w, r, f)
	if !ok {
		return
	}

	b.Lock()
	issue, err := b.agent.GetIssue(inum)
	b.Unlock()

	if err != nil {
		msg = fmt.Sprintf("Unable to find issue %d", inum)
		log.Info("Unable to find issue ", inum, ": ", err)
		return
	}

	if issue.Assignee == nil {
		assignee = ""
	} else {
		assignee = issue.Assignee.Login
		if s, ok := b.g2s[assignee]; ok {
			assignee = "@" + s
		}
		assignee = "\tAssigned to: " + assignee
	}

	msg = fmt.Sprintf("Issue %d: %q\n\tURL: %s\n\tState: %s\n%s", inum, issue.Title, issue.HTMLURL, issue.State, assignee)
}

func closeIssue(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "closeIssue")
	msg := "usage: /issue close NUMBER"
	defer func(){w.Write([]byte(msg))}()

	inum, ok := parseSimpleNumCmd(w, r, f)
	if !ok {
		return
	}

	// XXX TODO: make this a channel-wide announcement
	msg = fmt.Sprintf("Issue %d successfully closed", inum)

	b.Lock()
	err := b.agent.CloseIssue(inum)
	b.Unlock()

	if err != nil {
		msg = fmt.Sprintf("Unable to close issue %d", inum)
		log.Info("Unable to close issue ", inum, ": ", err)
	}
}

func reopenIssue(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "reopenIssue")
	msg := "usage: /issue reopen NUMBER"
	defer func(){w.Write([]byte(msg))}()

	inum, ok := parseSimpleNumCmd(w, r, f)
	if !ok {
		return
	}

	// XXX TODO: make this a channel-wide announcement
	msg = fmt.Sprintf("Issue %d successfully reopened", inum)
	b.Lock()
	err := b.agent.OpenIssue(inum)
	b.Unlock()

	if err != nil {
		msg = fmt.Sprintf("Unable to reopen issue %d", inum)
		log.Infof("Unable to reopen issue ", inum, ": ", err)
	}
}

func assignIssue(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "assignIssue")
	msg := "usage: /issue assign NUM [@SLACKNAME|@me|GITHUBNAME]"
	defer func(){w.Write([]byte(msg))}()

	if len(f) != 2 {
		return
	}
	inum, err := strconv.Atoi(f[0])
	if err != nil {
		return
	}

	b.Lock()
	defer b.Unlock()

	name := f[1]
	gname := name
	if gname == "@me" {
		name, err = getField("user_name", r)
		if err != nil {
			reqErr(log, w, err)
			return
		}
		name = "@" + name
	}
	if name[0] == '@' {
		s, ok := b.s2g[name[1:]]
		if !ok {
			msg = fmt.Sprintf("%q is not registered", name)
			return
		}
		gname = s
	}

	// XXX TODO: make this a channel-wide announcement
	msg = fmt.Sprintf("Issue %d is now assigned to %s", inum, name)
	err = b.agent.AssignIssue(inum, gname)

	if err != nil {
		msg = fmt.Sprintf("Unable to assign issue %d to %q", inum, name)
		log.Info("Unable to assign issue ", inum, " to ", gname, ": ", err)
	}
}

func unassignIssue(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "unassignIssue")
	msg := "usage: /issue unassign NUMBER"
	defer func(){w.Write([]byte(msg))}()

	inum, ok := parseSimpleNumCmd(w, r, f)
	if !ok {
		return
	}

	// XXX TODO: make this a channel-wide announcement
	msg = fmt.Sprintf("Issue %d is no longer assigned to anyone", inum)
	b.Lock()
	err := b.agent.UnassignIssue(inum)
	b.Unlock()

	if err != nil {
		msg = fmt.Sprintf("Unable to unassign issue %d", inum)
		log.Info("Unable to unassign issue ", inum, ": ", err)
	}
}

func registerUser(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "registerUser")
	msg := "usage: /issue register GITHUBUSER"
	defer func(){w.Write([]byte(msg))}()

	sname, err := getField("user_name", r)
	if err != nil {
		reqErr(log, w, err)
		return
	}
	if len(f) != 1 {
		return
	}

	msg = fmt.Sprintf("You are now registered as github user %q\n", f[0])
	b.Lock()
	defer b.Unlock()
	if !b.AddUserMap(sname, f[0]) {
		msg = fmt.Sprintf("Registration conflict")
	}
}

func getAlias(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "getAlias")
	msg := "usage: /issue get-alias"
	defer func(){w.Write([]byte(msg))}()

	sname, err := getField("user_name", r)
	if err != nil {
		reqErr(log, w, err)
		return
	}

	b.Lock()
	defer b.Unlock()
	gname, ok := b.s2g[sname]
	if !ok {
		msg = fmt.Sprintf("You are currently not registered as a github user")
	} else {
		msg = fmt.Sprintf("You are currently registered as github user %q", gname)
	}
}

func unregisterUser(b *IssueBot, w http.ResponseWriter, r *http.Request, f []string) {
	log := log.WithField("method", "unregisterUser")
	msg := "usage: /issue unregister"
	defer func(){w.Write([]byte(msg))}()

	sname, err := getField("user_name", r)
	if err != nil {
		reqErr(log, w, err)
		return
	}

	b.Lock()
	defer b.Unlock()
	b.DelUserBySlack(sname)
	msg = "Registration cleared"
}

// --------------------- UTILITY PARSING FUNCIONS ---------------------

func reqErr(log *logrus.Entry, w http.ResponseWriter, e error) {
	w.Write([]byte("Error:  malformed request"))
	log.Warn("Error processing request: ", e)
}

func getField(field string, r *http.Request) (string, error) {
	ss, ok := r.PostForm[field]
	if !ok || len(ss) > 1 {
		return "", fmt.Errorf("%q field missing or malformed in request", field)
	}
	return ss[0], nil

}

func parseSimpleNumCmd(w http.ResponseWriter, r *http.Request, f []string) (int, bool) {
	if len(f) != 1 {
		return -1, false
	}
	inum, err := strconv.Atoi(f[0])
	if err != nil {
		return -1, false
	}
	return inum, true
}

