package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TwinProduction/gatus/client"
	"io/ioutil"
)

type requestBody struct {
	Text string `json:"text"`
}

// SendMessage sends a message to the given Slack webhook
func SendMessage(webhookUrl, msg string) error {
	body, _ := json.Marshal(requestBody{Text: msg})
	response, err := client.GetHttpClient().Post(webhookUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	output, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body: %v", err.Error())
	}
	if string(output) != "ok" {
		return fmt.Errorf("error: %s", string(output))
	}
	return nil
}
