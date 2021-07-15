package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	commons "github.com/DAv10195/submit_commons"
	submitws "github.com/DAv10195/submit_commons/websocket"
	"net/http"
	"strings"
)

type outputRule func (string, map[string]interface{}) (string, error)

var outputRules map[string]outputRule

func readOutputLines(output string) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func mossOutputRule(output string, labels map[string]interface{}) (string, error) {
	lines, err := readOutputLines(output)
	if err != nil {
		return "", err
	}
	var mossLink string
	for _, line := range lines {
		if strings.HasPrefix(line, mossLinkPrefix) {
			mossLink = line
			break
		}
	}
	if mossLink == "" {
		return "", fmt.Errorf("mo moss link found in output: %s", output)
	}
	labels[commons.MossLink] = mossLink
	r, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:4567/parse?url=%s", mossLink), nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{}).Do(r)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.WithError(err).Errorf("error closing response from '%s'", mossLink)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error accessing moss parser. Response Status: %d instead of expected %d", resp.StatusCode, http.StatusOK)
	}
	mo := &submitws.MossOutput{}
	if err := json.NewDecoder(resp.Body).Decode(mo); err != nil {
		return "", err
	}
	for _, mop := range mo.Pairs {
		mop.Name1 = strings.TrimSuffix(mop.Name1, "/")
		mop.Name2 = strings.TrimSuffix(mop.Name2, "/")
	}
	mo.Link = mossLink
	outputBytes, err := json.Marshal(mo)
	if err != nil {
		return "", err
	}
	return string(outputBytes), nil
}

func init() {
	outputRules = make(map[string]outputRule)
	outputRules[commons.Moss] = mossOutputRule
}
