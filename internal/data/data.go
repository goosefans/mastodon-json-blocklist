package data

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Data represents the domain block data of a Mastodon instance
type Data struct {
	DomainBlocks []*DomainBlock `json:"domain_blocks"`
}

type BlockSeverity string

const (
	BlockSeveritySilence BlockSeverity = "silence"
	BlockSeveritySuspend BlockSeverity = "suspend"
	BlockSeverityNone    BlockSeverity = "none"
)

// DomainBlock represents a single domain block entry
type DomainBlock struct {
	Domains       []string      `json:"domains"`
	Severity      BlockSeverity `json:"severity"`
	RejectMedia   bool          `json:"reject_media"`
	RejectReports bool          `json:"reject_reports"`
	Reason        string        `json:"reason"`
}

// Retrieve retrieves and parses the domain block data currently available under the given URL
func Retrieve(url string) (*Data, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	parsed := new(Data)
	if err := json.Unmarshal(data, parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

// Sanitize sanitizes the domain block data using the following rules:
//   - Invalid domains will be removed silently
//   - Duplicate domains will be removed (the one defined the latest will be used)
//   - If an invalid/no severity was given, 'none' will be used
//   - RejectMedia and RejectReports will default to 'false' if not specified
func (data *Data) Sanitize() {
	usedDomains := make(map[string]bool, len(data.DomainBlocks))
	blocks := make([]*DomainBlock, 0, len(data.DomainBlocks))

	for i := len(data.DomainBlocks) - 1; i >= 0; i-- {
		block := data.DomainBlocks[i]

		domains := make([]string, 0, len(block.Domains))
		for _, domain := range block.Domains {
			domain = strings.TrimSpace(strings.ToLower(domain))
			if !strings.Contains(domain, ".") {
				continue
			}
			if !usedDomains[domain] {
				domains = append(domains, domain)
				usedDomains[domain] = true
			}
		}
		if len(block.Domains) == 0 {
			continue
		}

		if block.Severity != BlockSeveritySuspend && block.Severity != BlockSeveritySilence && block.Severity != BlockSeverityNone {
			block.Severity = BlockSeverityNone
		}

		blocks = append(blocks, block)
	}

	data.DomainBlocks = blocks
}
