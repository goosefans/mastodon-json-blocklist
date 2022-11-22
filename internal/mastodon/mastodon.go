package mastodon

import (
	"encoding/json"
	"fmt"
	"github.com/goosefans/mastodon-json-blocklist/internal/data"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type DomainBlock struct {
	ID            string `json:"id"`
	Domain        string `json:"domain"`
	Severity      string `json:"severity"`
	RejectMedia   bool   `json:"reject_media"`
	RejectReports bool   `json:"reject_reports"`
	PublicComment string `json:"public_comment"`
}

type Client struct {
	URL         string
	AccessToken string
}

func (client *Client) SyncData(raw *data.Data) error {
	raw.Sanitize()
	return client.syncInternalData(translateData(raw))
}

func (client *Client) getDomainBlocks() ([]*DomainBlock, error) {
	rUrl, err := url.Parse(client.URL + "/api/v1/admin/domain_blocks")
	if err != nil {
		return nil, err
	}
	request := &http.Request{
		URL: rUrl,
		Header: map[string][]string{
			"Authorization": {
				"Bearer " + client.AccessToken,
			},
		},
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var blocks []*DomainBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func (client *Client) createDomainBlock(block *DomainBlock) error {
	form := url.Values{}
	form.Add("domain", block.Domain)
	form.Add("severity", block.Severity)
	form.Add("reject_media", strconv.FormatBool(block.RejectMedia))
	form.Add("reject_reports", strconv.FormatBool(block.RejectReports))
	form.Add("public_comment", block.PublicComment)

	req, err := http.NewRequest(http.MethodPost, client.URL+"/api/v1/admin/domain_blocks", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+client.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (client *Client) updateDomainBlock(id string, block *DomainBlock) error {
	form := url.Values{}
	form.Add("severity", block.Severity)
	form.Add("reject_media", strconv.FormatBool(block.RejectMedia))
	form.Add("reject_reports", strconv.FormatBool(block.RejectReports))
	form.Add("public_comment", block.PublicComment)

	req, err := http.NewRequest(http.MethodPut, client.URL+"/api/v1/admin/domain_blocks/"+id, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+client.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (client *Client) deleteDomainBlock(id string) error {
	req, err := http.NewRequest(http.MethodDelete, client.URL+"/api/v1/admin/domain_blocks/"+id, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+client.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (client *Client) syncInternalData(newState []*DomainBlock) error {
	// Retrieve all current domain blocks and map them to their domain
	currentState, err := client.getDomainBlocks()
	if err != nil {
		return err
	}
	currentDomainMap := make(map[string]*DomainBlock, len(currentState))
	for _, block := range currentState {
		currentDomainMap[block.Domain] = block
	}

	for _, newBlock := range newState {
		// Create the new domain block if the domain has no current block mapped to it
		currentBlock, exists := currentDomainMap[newBlock.Domain]
		if !exists {
			log.Info().Str("domain", newBlock.Domain).Msgf("Creating domain block for '%s'...", newBlock.Domain)
			if err := client.createDomainBlock(newBlock); err != nil {
				return err
			}
			continue
		}

		// Update the domain block if it differs from the current one
		if differs(currentBlock, newBlock) {
			log.Info().Str("domain", newBlock.Domain).Msgf("Updating domain block for '%s'...", newBlock.Domain)
			if err := client.updateDomainBlock(currentBlock.ID, newBlock); err != nil {
				return err
			}
		}

		// We reuse the domain -> current block map later on to delete domain blocks removed from the data.
		// As this domain block existed in the new data set, we don't want to delete it.
		delete(currentDomainMap, newBlock.Domain)
	}

	// Delete every remaining current domain block
	for _, removedBlock := range currentDomainMap {
		log.Info().Str("domain", removedBlock.Domain).Msgf("Removing domain block for '%s'...", removedBlock.Domain)
		if err := client.deleteDomainBlock(removedBlock.ID); err != nil {
			return err
		}
	}
	return nil
}

func differs(a, b *DomainBlock) bool {
	return a.Severity != b.Severity || a.RejectMedia != b.RejectMedia || a.RejectReports != b.RejectReports || a.PublicComment != b.PublicComment
}

func translateData(raw *data.Data) []*DomainBlock {
	blocks := make([]*DomainBlock, 0, len(raw.DomainBlocks))
	for _, rawBlock := range raw.DomainBlocks {
		for _, domain := range rawBlock.Domains {
			blocks = append(blocks, &DomainBlock{
				Domain:        domain,
				Severity:      string(rawBlock.Severity),
				RejectMedia:   rawBlock.RejectMedia,
				RejectReports: rawBlock.RejectReports,
				PublicComment: rawBlock.Reason,
			})
		}
	}
	return blocks
}
