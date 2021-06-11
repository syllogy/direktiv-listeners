package utils

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

// SendCloudEvent sends cloud events to direktiv
func SendCloudEvent(url, token string) error {

	// u := fmt.Sprintf("%s/api/namespaces/%s/event", DirektivURL, DirektivNamespace)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		log.Printf("failed to create event request: %s", err.Error())
		return
	}
	defer req.Body.Close()

	if len(token) > 0 {
		req.Header.Set("Authorization", token)
		// req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", DirektivToken))
	}

	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("%+v\n", req)

	// resp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	log.Printf("failed to send request: %s", err.Error())
	// 	return
	// }
	// defer resp.Body.Close()
	//
	// if resp.StatusCode >= 300 {
	// 	log.Printf("postEvent returned status code %d", resp.StatusCode)
	// 	return
	// }

	return nil
}
