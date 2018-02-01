// Dumb test slackbot
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ctelfer-docker/slkiss/slack"
)

var addr  = flag.String("l", "", "Address to listen on")
var port  = flag.Uint("p", 80, "Port to listen on")
var cfgfn = flag.String("c", "", "Config field to load")

func main() {
	flag.Parse()
	a := fmt.Sprintf("%s:%d", *addr, *port)
	log.Println(a)
	bot := slack.NewIssueBot(a, "ctelfer-docker/slkiss")
	bot.AddUserMap("ctelfer", "ctelfer-docker")
	if *cfgfn != "" {
		log.Println("Loading config file")
	}
	log.Println("Starting bot")
	bot.Run()
}
