// Manages github issues using slash commands in slack
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ctelfer-docker/slkiss/slack"
)

var repo  = flag.String("r", "ctelfer-docker/slkiss", "Default repository to manage")
var user  = flag.String("u", "ctelfer-docker", "User for the bot to operate as")
var auth  = flag.String("a", "", "Authentication token")
var addr  = flag.String("l", "", "Address to listen on")
var port  = flag.Uint("p", 80, "Port to listen on")
var cfgfn = flag.String("c", "", "Config field to load")

const authEnv = "GHMOD_PASSWORD"

func main() {
	flag.Parse()
	if *auth == "" {
		s, ok := os.LookupEnv(authEnv)
		if !ok {
			usage()
		}
		*auth = s
	}
	astr := fmt.Sprintf("%s:%d", *addr, *port)
	bot := slack.NewIssueBot(astr, *repo)
	bot.SetGithubAuth(encodeBasicAuth(*user, *auth))
	loadConfig(bot)
	log.Println("Starting bot")
	bot.Run()
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\t* Authentication token is required unless set in GHMOD_PASSWORD\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func encodeBasicAuth(u string, pw string) string {
	s := u + ":" + pw
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	return "Basic " + b64
}

func loadConfig(bot *slack.IssueBot) {
	if *cfgfn == "" {
		return
	}
	log.Println("Loading config file")
	// TODO
}
