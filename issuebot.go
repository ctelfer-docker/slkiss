// Manages github issues using slash commands in slack
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/ctelfer-docker/slkiss/slack"
	"github.com/sirupsen/logrus"
)

// Environment variables
const (
	repoEnv = "ISSUEBOT_REPO"     // Github repository to manage
	userEnv = "ISSUEBOT_USER"     // Github user to access the repo
	authEnv = "ISSUEBOT_AUTH"     // Authentication token for Github
	addrEnv = "ISSUEBOT_LADDR"    // Local address to bind to for slack ops 
	portEnv = "ISSUEBOT_LPORT"    // Local port to bind to for slack ops
	logEnv  = "ISSUEBOT_LOGLEVEL" // Log level to run at
)

// Name so that *Level will implement flag.Value type
type Level logrus.Level

// Implement flag.Value.
func (l *Level) String() string {
	return logrus.Level(*l).String()
}

// Implement flag.Value.
func (l *Level) Set(level string) error {
	nl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	*l = Level(nl)
	return nil
}

// CLI argumetnts
var repo  = flag.String("r", "", "Default repository to manage")
var user  = flag.String("u", "", "Github user for the bot to operate as")
var auth  = flag.String("a", "", "Authentication token")
var addr  = flag.String("l", "", "Address to listen on")
var port  = flag.Uint("p", 80, "Port to listen on")
var logLevel = Level(logrus.InfoLevel)

func init() {
	flag.Var(&logLevel, "L", "Set the log level for the daemon")
}

func main() {
	getEnv()
	flag.Parse()
	if *repo == "" || *user == "" || *auth == "" {
		usage()
	}
	logrus.SetLevel(logrus.Level(logLevel))
	astr := fmt.Sprintf("%s:%d", *addr, *port)
	bot := slack.NewIssueBot(astr, *repo)
	bot.SetGithubAuth(encodeBasicAuth(*user, *auth))
	logrus.Info("Starting bot on", astr)
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
			logrus.Fatal("Error with local port:", err)
		}
		*port = uint(p)
	}
	if s, ok := os.LookupEnv(logEnv); ok {
		err := (&logLevel).Set(s)
		if err != nil {
			logrus.Fatal("Error setting log level:", err)
		}
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
