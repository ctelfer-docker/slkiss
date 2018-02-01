package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/ctelfer-docker/slkiss/github"
)

var repo = flag.String("r", "ctelfer-docker/slkiss", "Default repository to search")
var inum = flag.Int("i", -1, "Issue number to fetch")
var user = flag.String("u", "ctelfer-docker", "User to operate as")
var auth = flag.String("a", "", "Authentication token")

const authEnv = "GHMOD_PASSWORD"

func encodeBasicAuth(u string, pw string) string {
	s := u + ":" + pw
	b64 := base64.StdEncoding.EncodeToString([]byte(s))
	return "Basic " + b64
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [OPT] [open|close|unassign|assign USER]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\t* Issue token is requred\n")
	fmt.Fprintf(os.Stderr, "\t* Authentication token is required unless set in GHMOD_PASSWORD\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var err error

	flag.Parse()
	if *inum < 0 || flag.NArg() < 1 {
		usage()
	}

	if *auth == "" {
		s, ok := os.LookupEnv(authEnv)
		if !ok {
			usage()
		}
		*auth = s
	}

	a := github.NewRepoAgent(*repo)
	t := encodeBasicAuth(*user, *auth)
	a.SetToken(t)

	switch flag.Arg(0) {
	case "open":
		err = a.OpenIssue(*inum)
	case "close":
		err = a.CloseIssue(*inum)
	case "assign":
		if flag.NArg() != 2 {
			usage()
		}
		err = a.AssignIssue(*inum, flag.Arg(1))
	case "unassign":
		err = a.UnassignIssue(*inum)
	default:
		usage()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println("Operation completed successfully")
}
