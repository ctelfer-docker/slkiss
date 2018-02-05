// Manages github issues using slash commands in slack
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ctelfer-docker/slkiss/slack"
)

var repo  = flag.String("r", "", "Default repository to manage")
var user  = flag.String("u", "", "Github user for the bot to operate as")
var auth  = flag.String("a", "", "Authentication token")
var addr  = flag.String("l", "", "Address to listen on")
var port  = flag.Uint("p", 80, "Port to listen on")

const (
	repoEnv = "ISSUEBOT_REPO"
	userEnv = "ISSUEBOT_USER"
	authEnv = "ISSUEBOT_AUTH"
	addrEnv = "ISSUEBOT_LADDR"
	portEnv = "ISSUEBOT_LPORT"
)

func main() {
	getEnv()
	flag.Parse()
	if *repo == "" || *user == "" || *auth == "" {
		usage()
	}
	astr := fmt.Sprintf("%s:%d", *addr, *port)
	bot := slack.NewIssueBot(astr, *repo)
	bot.SetGithubAuth(encodeBasicAuth(*user, *auth))
	log.Println("Starting bot")
	bot.Run()
}

func getEnv() {
	if s, ok := os.LookupEnv(repoEnv); ok { *repo = s }
	if s, ok := os.LookupEnv(userEnv); ok { *user = s }
	if s, ok := os.LookupEnv(authEnv); ok { *auth = s }
	if s, ok := os.LookupEnv(addrEnv); ok { *addr = s }
	if s, ok := os.LookupEnv(portEnv); ok {
		p, err := strconv.Atoi(s)
		if err != nil {
			log.Fatal("Error with local port:", err)
		}
		*port = uint(p)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "-----\n")
	fmt.Fprintf(os.Stderr, "\t* User, Repo and Authentication token are all required\n")
	fmt.Fprintf(os.Stderr, "\t* One can also set these options via environment variables:\n")
	fmt.Fprintf(os.Stderr, "\t*   %s - repository\n", repoEnv)
	fmt.Fprintf(os.Stderr, "\t*   %s - github user\n", userEnv)
	fmt.Fprintf(os.Stderr, "\t*   %s - github authentication password\n", authEnv)
	fmt.Fprintf(os.Stderr, "\t*   %s - local address\n", addrEnv)
	fmt.Fprintf(os.Stderr, "\t*   %s - local port\n", portEnv)
	os.Exit(1)
}

func encodeBasicAuth(u string, pw string) string {
	s := u + ":" + pw
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	return "Basic " + b64
}
