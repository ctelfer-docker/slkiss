// Tracks github issues using github's API
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var _ = ioutil.ReadAll // XXX REMOVE ME when done debugging

const BaseURL = "https://api.github.com/repos/"

// This represents the fields of an individual issue.
type Issue struct {
	Number    int
	Title     string
	ID        int
	HTMLURL   string `json:"html_url"`
	State     string
	User      *User
	Assignee  *User
	Body      string
	CreatedAt time.Time `json:created_at`
	UpdatedAt time.Time `json:updated_at`
	ClosedAt  time.Time `json:closed_at`
	Labels    []*Label
	Locked    bool
}

// This type represents a github user entry
type User struct {
	Login   string
	ID      int
	HTMLURL string `json:"html_url"`
}

// This type represents a github issue Label
type Label struct {
	Name string
	URL  string
}

// This is a helper function that issues a github issue query and
// returns the full results or an error.  Github limits the number
// of entries per 'GET' so thsi function invokes getLink() to find the
// link to the next set of results for the query.
func SearchIssues(base string, params map[string]string) ([]*Issue, error) {
	var result []*Issue
	p := ""
	for k, v := range params {
		if p == "" {
			p = "?" + url.QueryEscape(k) + "=" + url.QueryEscape(v)
		} else {
			p += "&" + url.QueryEscape(k) + "=" + url.QueryEscape(v)
		}
	}
	fmt.Println(base + p)
	resp, err := http.Get(base + p)
	if err != nil {
		return nil, err
	}
	if result, err = decodeIssueList(result, resp); err != nil {
		return nil, err
	}
	for link := getLink(resp); link != ""; link = getLink(resp) {
		if resp, err = http.Get(link); err != nil {
			return nil, err
		}
		if result, err = decodeIssueList(result, resp); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Fetch a particular issue from github based on its number.
// This function assumes that `base` includes a full repo path for a query.
//   e.g. https://api.github.com/repos/OWNER/REPO
//
func GetIssue(base string, num int) (*Issue, error) {
	var iss Issue
	resp, err := http.Get(base + fmt.Sprintf("/%d", num))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&iss); err != nil {
		return nil, err
	}
	return &iss, nil
}

// Modify a github issue.
//
// The map argument is going to get encoded into a JSON request to send
// directly to the github API.  See:
//   https://developer.github.com/v3/issues/#edit-an-issue
//
func ModIssue(base string, tok string, num int, fields map[string]interface{}) error {
	if tok == "" {
		return fmt.Errorf("Token required for ModIssue")
	}

	addr := base + fmt.Sprintf("/%d", num)

	json, err := json.Marshal(fields)
	if err != nil {
		fmt.Errorf("Error marshalling request: %s", err.Error())
	}

	fmt.Println("JSON: ", string(json))
	req, err := http.NewRequest(http.MethodPatch, addr, bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", tok)

	client := http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()

	// REMOVEME
	if err == nil {
		fmt.Println("Response Status:", resp.Status)
		fmt.Println("Response Headers:\n", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("Response Body:\n", string(body))
	}
	// REMOVEME

	return err
}

// Github Link Format
//
// <https://api.github.com/repositories/31046054/issues?page=1>; rel="prev",
//   <https://api.github.com/repositories/31046054/issues?page=3>; rel="next",
//   <https://api.github.com/repositories/31046054/issues?page=7>; rel="last",
//   <https://api.github.com/repositories/31046054/issues?page=1>; rel="first"
//
// Where these are on one line and any of these can be omitted.
//
const nextREStr = `<([^>]+)>; rel="next"`

var nextRE = regexp.MustCompile(nextREStr)

// This function searches for a Link: field in the github issue response header
// containing a "next" entry and returns the URL associated with that entry if
// it is present.
func getLink(resp *http.Response) string {
	var links = resp.Header["Link"]
	if links == nil {
		return ""
	}
	s := links[0]
	matches := nextRE.FindStringSubmatch(s)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

// This function decodes the JSON response and appends any issues found
// therein into the slice of issues
func decodeIssueList(issues []*Issue, resp *http.Response) ([]*Issue, error) {
	defer resp.Body.Close()
	var di []*Issue
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search query failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&di); err != nil {
		return nil, err
	}
	return append(issues, di...), nil

}

// This struct is for convenience when issuing multiple queries to the
// same repo with (roughty) the same set of base query parameters.
type Agent struct {
	base        string
	token       string
	fixedParams map[string]string
}

// This function is a constructor for a generic github issue searcher
func NewAgent(base string, params map[string]string) *Agent {
	return &Agent{base, "", params}
}

// This function is a constructor for a github issue searcher that
// always queries issues from a specific owner/repo.
func NewRepoAgent(name string) *Agent {
	return NewAgent(BaseURL+name+"/issues", make(map[string]string))
}

// This function adds search parameters to the fixed parameters for the searcher.
func (s *Agent) AddParam(key, value string) {
	s.fixedParams[key] = value
}

// This function queries for a set of github issues matching a given set of
// parameters.  The user-specified parameters override the Agent's
// fixed paramters if the two parameter sets overlap.
func (s *Agent) FetchIssues(params map[string]string) ([]*Issue, error) {
	p := make(map[string]string)
	for k, v := range s.fixedParams {
		p[k] = v
	}
	for k, v := range params {
		p[k] = v
	}
	return SearchIssues(s.base, p)
}

// Set the authentication token for a given agent.
func (s *Agent) SetToken(token string) {
	s.token = token
}

// Read a specific issue by its issue number.
func (s *Agent) GetIssue(num int) (*Issue, error) {
	return GetIssue(s.base, num)
}

// Modify an issue in some way.  See ModIssue()
//
// Other methods will build higher level changes on top of this.
func (s *Agent) modIssue(num int, m map[string]interface{}) error {
	return ModIssue(s.base, s.token, num, m)
}

// Close an existing issue
func (s *Agent) CloseIssue(num int) error {
	return s.modIssue(num, map[string]interface{}{"state": "closed"})
}

// [Re]Open an existing issue
func (s *Agent) OpenIssue(num int) error {
	return s.modIssue(num, map[string]interface{}{"state": "open"})
}

// Remove assigned users from this issue
func (s *Agent) UnassignIssue(num int) error {
	ulist := []string{}
	return s.modIssue(num, map[string]interface{}{"assignees": ulist})
}

// Assign a user to this issue
func (s *Agent) AssignIssue(num int, user string) error {
	ulist := []string{user}
	return s.modIssue(num, map[string]interface{}{"assignees": ulist})
}
