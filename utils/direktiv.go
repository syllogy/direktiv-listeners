package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

const (
	direktivURL   = "DIREKTIV_URL"
	direktivToken = "DIREKTIV_TOKEN"
)

// SendCloudEvent sends cloud events to direktiv
func SendCloudEvent(event *cloudevents.Event) error {

	b, err := json.Marshal(event)
	if err != nil {
		return err
	}

	url := os.Getenv(direktivURL)
	token := os.Getenv(direktivToken)

	if url == "" {
		return fmt.Errorf("no direktiv url provided")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer req.Body.Close()

	if len(token) > 0 {
		req.Header.Set("Authorization", token)
	}

	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("%+v\n", req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("postEvent returned status code %d", resp.StatusCode)
	}

	return nil
}
