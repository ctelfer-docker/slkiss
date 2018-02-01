// This package contains an issue managing slackbot that
// listens to slash commands on HTTP ports, invokes github
// queryies in response and reports on the results.
package slack

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"strconv"
	"sync"

	"github.com/ctelfer-docker/slkiss/github"
)

var handlers = []struct{
	pat string
	f botHandlerFunc
}{
	{"/find-issue", findIssue},
	{"/close-issue", closeIssue},
	{"/reopen-issue", reopenIssue},
	{"/assign-issue", assignIssue},
	{"/unassign-issue", unassignIssue},
	{"/register", registerUser},
	{"/getalias", getAlias},
	{"/", invalidOp},

}

// Bot implements a slackbot that manages 
type IssueBot struct {
	sync.Mutex
	addr   string
	mux    *http.ServeMux
	agent  *github.Agent
	g2s    map[string]string
	s2g    map[string]string
}

type botHandlerFunc func(*IssueBot, http.ResponseWriter, *http.Request)

type botHandlerCtx struct {
	b *IssueBot
	f botHandlerFunc
}

func (c *botHandlerCtx)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.f(c.b, w, r)
}

// Create a new IssueBot
func NewIssueBot(addr string, repo string) *IssueBot {
	b := &IssueBot{}
	b.addr = addr
	b.mux = http.NewServeMux()
	b.agent = github.NewRepoAgent(repo)
	for _, h := range handlers {
		b.addHandler(h.pat, h.f)

	}
	b.g2s = make(map[string]string)
	b.s2g = make(map[string]string)
	return b
}

func (b *IssueBot) SetGithubAuth(token string) {
	b.agent.SetToken(token)
}

func (b *IssueBot) addHandler(pattern string, f botHandlerFunc) {
	c := &botHandlerCtx{b, f}
	b.mux.Handle(pattern, c)
}

// Add a mapping from a slack username (sname) to a github username (gname).
func (b *IssueBot) AddUserMap(sname string, gname string) {
	b.Lock()
	defer b.Unlock()
	b.g2s[gname] = sname
	b.s2g[sname] = gname
}

// Delete the name mappings for the user specified by the slack name sname.
func (b *IssueBot) DelUserBySlack(sname string) {
	b.Lock()
	defer b.Unlock()
	gname, ok := b.s2g[sname]
	if ok {
		delete(b.g2s, gname)
		delete(b.s2g, sname)
	}

}

func (b *IssueBot) Run() {
	log.Fatal(http.ListenAndServe(b.addr, b.mux))
}

func findIssue(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	var assignee string
	inum, ok := parseSimpleNumCmd(w, r, "usage: /find-issue NUMBER")
	if !ok {
		return
	}
	issue, err := b.agent.GetIssue(inum)
	if err != nil {
		msg := fmt.Sprintf("Unable to find issue %d", inum)
		log.Println("Unable to find issue", inum, ":", err)
		w.Write([]byte(msg))
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
	msg := fmt.Sprintf("Issue %d: %q\n\tURL: %s\n\tState: %s\n%s", inum, issue.Title, issue.HTMLURL, issue.State, assignee)
	w.Write([]byte(msg))
}

func closeIssue(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	inum, ok := parseSimpleNumCmd(w, r, "usage: /close-issue NUMBER")
	if !ok {
		return
	}
	// XXX TODO: make this a channel-wide announcement
	msg := fmt.Sprintf("Issue %d successfully closed", inum)
	err := b.agent.CloseIssue(inum)
	if err != nil {
		msg = fmt.Sprintf("Unable to close issue %d", inum)
		log.Println("Unable to close issue", inum, ":", err)
	}
	w.Write([]byte(msg))
}

func reopenIssue(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	inum, ok := parseSimpleNumCmd(w, r, "usage: /reopen-issue NUMBER")
	if !ok {
		return
	}
	// XXX TODO: make this a channel-wide announcement
	msg := fmt.Sprintf("Issue %d successfully reopened", inum)
	err := b.agent.OpenIssue(inum)
	if err != nil {
		msg = fmt.Sprintf("Unable to reopen issue %d", inum)
		log.Println("Unable to reopen issue", inum, ":", err)
	}
	w.Write([]byte(msg))
}

func assignIssue(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	usageMsg := []byte("usage: /assign-issue NUM [@SLACKNAME|@me|GITHUBNAME]")
	if err := r.ParseForm(); err != nil {
		reqErr(w, err)
		return
	}
	text, err := getField("text", r)
	if err != nil {
		reqErr(w, err)
		return
	}
	ss := strings.Fields(text)
	if len(ss) != 2 {
		w.Write(usageMsg)
		return
	}
	inum, err := strconv.Atoi(ss[0])
	if err != nil {
		w.Write(usageMsg)
	}
	name := ss[1]
	gname := name
	if gname == "@me" {
		name, err = getField("user_name", r)
		if err != nil {
			reqErr(w, err)
			return
		}
		name = "@" + name
	}
	if name[0] == '@' {
		s, ok := b.s2g[name[1:]]
		if !ok {
			msg := fmt.Sprintf("%q is not registered", name)
			w.Write([]byte(msg))
			return
		}
		gname = s
	}
	// XXX TODO: make this a channel-wide announcement
	msg := fmt.Sprintf("Issue %d is now assigned to %s", inum, name)
	err = b.agent.AssignIssue(inum, gname)
	if err != nil {
		msg = fmt.Sprintf("Unable to assign issue %d to %q", inum, name)
		log.Println("Unable to assign issue", inum, "to", gname, ":", err)
	}
	w.Write([]byte(msg))
}

func unassignIssue(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	inum, ok := parseSimpleNumCmd(w, r, "usage: /unassign-issue NUMBER")
	if !ok {
		return
	}
	// XXX TODO: make this a channel-wide announcement
	msg := fmt.Sprintf("Issue %d is no longer assigned to anyone", inum)
	err := b.agent.UnassignIssue(inum)
	if err != nil {
		msg = fmt.Sprintf("Unable to unassign issue %d", inum)
		log.Println("Unable to unassign issue", inum, ":", err)
	}
	w.Write([]byte(msg))
}

func registerUser(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		reqErr(w, err)
		return
	}
	sname, err := getField("user_name", r)
	if err != nil {
		reqErr(w, err)
		return
	}
	gname, err := getField("text", r)
	if err != nil {
		reqErr(w, err)
		return
	}
	gname = strings.Trim(gname, " \r\n")
	b.AddUserMap(sname, gname)
	w.Write([]byte(fmt.Sprintf("You are now registered as github user %q\n", gname)))
}

func getAlias(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		reqErr(w, err)
		return
	}
	sname, err := getField("user_name", r)
	if err != nil {
		reqErr(w, err)
		return
	}
	gname, ok := b.s2g[sname]
	if !ok {
		w.Write([]byte(fmt.Sprintf("You are currently not registered as a github user")))
	} else {
		w.Write([]byte(fmt.Sprintf("You are currently registered as github user %q", gname)))
	}
}

func invalidOp(b *IssueBot, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Invalid operation"))
}

func reqErr(w http.ResponseWriter, e error) {
	w.Write([]byte("Error:  malformed request"))
	log.Println("Error processing request:", e)
}

func getField(field string, r *http.Request) (string, error) {
	ss, ok := r.PostForm[field]
	if !ok || len(ss) > 1 {
		return "", fmt.Errorf("%q field missing or malformed in request", field)
	}
	return ss[0], nil

}

func parseSimpleNumCmd(w http.ResponseWriter, r *http.Request, usage string) (int, bool) {
	if err := r.ParseForm(); err != nil {
		reqErr(w, err)
		return -1, false
	}
	inumstr, err := getField("text", r)
	if err != nil {
		reqErr(w, err)
		return -1, false
	}
	inum, err := strconv.Atoi(inumstr)
	if err != nil {
		w.Write([]byte(usage))
		return -1, false
	}
	return inum, true
}

