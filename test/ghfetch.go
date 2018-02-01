package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/ctelfer-docker/slkiss/github"
)

var repo = flag.String("r", "ctelfer-docker/slkiss", "Default repository to search")
var inum = flag.Int("n", -1, "Issue number to fetch")

const issueTmpl = `Number:    {{.Number}}
Title:     {{.Title}}
Reporter:  {{.User.Login}}({{.User.ID}})
URL:       {{.HTMLURL}}
State:     {{.State}}
`

const listTmpl = `{{range .}}-----------------------------------------
` + issueTmpl + `{{end}}`

var issueRpt = template.Must(template.New("issue").Parse(issueTmpl))
var listRpt = template.Must(template.New("issueList").Parse(listTmpl))

func addParam(pm map[string]string, p string) {
	kv := strings.Split(p, "=")
	if len(kv) != 2 {
		log.Fatal(fmt.Errorf("Invalid parameter format: %q", p))
	}
	pm[kv[0]] = kv[1]
}

func main() {
	flag.Parse()

	a := github.NewRepoAgent(*repo)

	if *inum < 0 {
		a.AddParam("per_page", "100")
		pm := make(map[string]string)
		for _, s := range flag.Args() {
			addParam(pm, s)
		}
		issues, err := a.FetchIssues(pm)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("There are %d issues in the query\n", len(issues))
		if err := listRpt.Execute(os.Stdout, issues); err != nil {
			log.Fatal(err)
		}
	} else {
		if len(flag.Args()) > 0 {
			log.Fatal("Extra query parameters illegal when fetching one issue")
		}
		issue, err := a.GetIssue(*inum)
		if err != nil {
			log.Fatal(err)
		}
		if err = issueRpt.Execute(os.Stdout, issue); err != nil {
			log.Fatal(err)
		}
	}
}
