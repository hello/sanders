package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type BasicNotifier interface {
	Notify(action *DeployAction) error
}

func NewSlackNotifier(username string) *SlackNotifier {
	return &SlackNotifier{username: username, endpoint: "https://hooks.slack.com/services/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"}
}

type SlackNotifier struct {
	username string
	endpoint string
}

type Payload struct {
	Text        string       `json:"text"`
	Username    string       `json:"username"`
	IconEmoji   string       `json:"icon_emoji"`
	MarkDown    bool         `json:"mrkdwn"`
	Attachments []Attachment `json:"attachments"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type Attachment struct {
	Fallback   string  `json:"fallback"`
	Color      string  `json:"color"`
	AuthorName string  `json:"author_name"`
	Title      string  `json:"title"`
	Text       string  `json:"text"`
	Fields     []Field `json:"fields"`
}

func (n SlackNotifier) Notify(action *DeployAction) error {

	actionColors := make(map[string]string)
	actionColors["deploy"] = "good"
	actionColors["confirm"] = "good"
	actionColors["canary"] = "good"
	actionColors["create"] = "#764FA5"
	actionColors["sunset"] = "warning"

	fields := []Field{
		Field{Title: "App", Value: action.AppName, Short: true},
		Field{Title: "Version", Value: action.LC, Short: true},
		Field{Title: "Type", Value: action.CmdType, Short: true},
		Field{Title: "# servers", Value: fmt.Sprintf("%d", action.NumServers), Short: true},
	}

	singleAttachment := Attachment{
		AuthorName: n.username,
		Fields:     fields,
		Color:      actionColors[action.CmdType],
		Fallback: 	action.FallbackString(),
	}

	payload := &Payload{
		Username:    "sanders",
		IconEmoji:   ":rocket:",
		MarkDown:    true,
		Attachments: []Attachment{singleAttachment},
	}

	buff, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
		return err
	}

	req, _ := http.NewRequest("POST", n.endpoint, bytes.NewBuffer(buff))
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println(err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Println("Unexpected: %d", resp.StatusCode)
	}
	return nil
}

type DeployAction struct {
	CmdType    string
	AppName    string
	LC         string
	NumServers int64
}

func NewDeployAction(cmdType, appName, lc string, numServers int64) *DeployAction {
	return &DeployAction{
		CmdType:    cmdType,
		AppName:    appName,
		LC:         lc,
		NumServers: numServers,
	}
}

func (d *DeployAction) String() string {
	return fmt.Sprintf(`Type: *%s*
AppName: *%s*
LC: *%s*
NumServers: *%d*`, d.CmdType, d.AppName, d.LC, d.NumServers)
}

func (d *DeployAction) FallbackString() string {
	return fmt.Sprintf(`%s App: %s
LC: %s
NumServers: %d`, d.CmdType, d.AppName, d.LC, d.NumServers)
}
