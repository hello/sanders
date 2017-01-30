package core

import (
	"errors"
	"fmt"
	"github.com/mitchellh/cli"
	"github.com/sourcegraph/go-papertrail/papertrail"
	"time"
)

type Tailor struct {
	Ui cli.ColoredUi
}

func (t *Tailor) Run(systemID, query string) error {

	token, err := papertrail.ReadToken()
	if err == papertrail.ErrNoTokenFound {
		return errors.New("No Papertrail API token found; exiting.\n\npapertrail-go requires a valid Papertrail API token (which you can obtain from https://papertrailapp.com/user/edit) to be set in the PAPERTRAIL_API_TOKEN environment variable or in ~/.papertrail.yml (in the format `token: MYTOKEN`).")
	} else if err != nil {
		return err
	}

	client := papertrail.NewClient((&papertrail.TokenTransport{Token: token}).Client())

	t.Ui.Info("Tailing: " + systemID)

	opt := papertrail.SearchOptions{
		SystemID: systemID,
		GroupID:  "",
		Query:    query,
	}

	delay := 2 * time.Second
	opt.MinTime = time.Now().In(time.UTC).Add(-1 * time.Hour)

	stopWhenEmpty := false
	polling := false

	for {
		searchResp, httpResp, err := client.Search(opt)
		if httpResp.StatusCode == 404 {
			t.Ui.Info("Instance not ready, sleeping…")
			time.Sleep(delay * 2)
			continue
		}
		if searchResp == nil || err != nil {
			return errors.New(fmt.Sprintf("Invalid token? %s: %s", token, err))
		}

		if httpResp.StatusCode != 200 {
			return errors.New(fmt.Sprintf("Got http: %d", httpResp.StatusCode))
		}

		if len(searchResp.Events) == 0 {
			if stopWhenEmpty {
				return nil
			} else {
				// No more messages are immediately available, so now we'll just
				// poll periodically.
				polling = true
			}
		}

		for _, e := range searchResp.Events {
			t.Ui.Output(e.Message)
		}

		opt.MinID = searchResp.MaxID

		if polling {
			time.Sleep(delay)
		}
	}
}
