// Package cost computes session cost for the cost segment.
//
// See DESIGN.md §7. Two paths:
//   - API users: session.Cost.TotalCostUSD > 0 ⇒ use it directly ($X.XX).
//   - Subscription users: that field is 0 ⇒ estimate by parsing the transcript
//     JSONL: sum per-message token usage × a pricing table keyed by model
//     family. Render as "~$X.XX". Cache 10s keyed by transcript path hash.
//
// The pricing table is DATA (pricing.json), not code.
package cost

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/thissayantan/vitals/internal/cache"
	"github.com/thissayantan/vitals/internal/claude"
)

//go:embed pricing.json
var pricingData []byte

// Estimate is the resolved cost for a session.
type Estimate struct {
	USD       float64
	Estimated bool // true when derived from the transcript (subscription mode)
}

// Pricing is the per-1M-token rate for a model family.
type Pricing struct {
	In         float64 `json:"in"`
	Out        float64 `json:"out"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type pricingRule struct {
	Match []string `json:"match"`
	Pricing
}

type pricingTable struct {
	Default Pricing       `json:"default"`
	Rules   []pricingRule `json:"rules"`
}

var table pricingTable

func init() {
	// Pricing is embedded and trusted; ignore the unlikely parse error and fall
	// back to a hardcoded sonnet default.
	if err := json.Unmarshal(pricingData, &table); err != nil {
		table = pricingTable{Default: Pricing{In: 3, Out: 15, CacheRead: 0.3, CacheWrite: 3.75}}
	}
}

// priceFor returns the pricing for a model id by first-substring-match, falling
// back to the default.
func priceFor(model string) Pricing {
	m := strings.ToLower(model)
	for _, r := range table.Rules {
		for _, sub := range r.Match {
			if strings.Contains(m, sub) {
				return r.Pricing
			}
		}
	}
	return table.Default
}

// Get resolves the cost for s according to mode. Preferred values:
//   - "api":          always the platform-reported cost.total_cost_usd (actual).
//   - "subscription": always marked estimated; keeps the reported number when
//     present (Claude reports an API-equivalent cost even on subscription),
//     else falls back to the transcript estimate.
//   - "auto":         actual when total_cost_usd > 0, else the transcript estimate.
//
// Legacy aliases: "cc" == "api"; "estimate" == always the transcript estimate.
func Get(s *claude.Session, c *cache.Store, mode string) Estimate {
	switch mode {
	case "api", "cc":
		return Estimate{USD: s.Cost.TotalCostUSD, Estimated: false}
	case "estimate":
		return Estimate{USD: estimate(s, c), Estimated: true}
	case "subscription":
		if s.Cost.TotalCostUSD > 0 {
			return Estimate{USD: s.Cost.TotalCostUSD, Estimated: true}
		}
		return Estimate{USD: estimate(s, c), Estimated: true}
	default: // auto
		if s.Cost.TotalCostUSD > 0 {
			return Estimate{USD: s.Cost.TotalCostUSD, Estimated: false}
		}
		return Estimate{USD: estimate(s, c), Estimated: true}
	}
}

// estimate parses the transcript and sums per-message token cost, cached 10s.
func estimate(s *claude.Session, c *cache.Store) float64 {
	if s.TranscriptPath == "" {
		return 0
	}
	key := "cost:" + s.TranscriptPath
	out := c.Memo(key, cache.TTLCost, func() []byte {
		v := estimateFromTranscript(s.TranscriptPath)
		return []byte(strconv.FormatFloat(v, 'f', 4, 64))
	})
	v, _ := strconv.ParseFloat(string(out), 64)
	return v
}

// transcriptLine is the minimal shape we read from each JSONL row.
type transcriptLine struct {
	Type    string `json:"type"`
	Message struct {
		Model string `json:"model"`
		Usage *struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// estimateFromTranscript sums cost across all assistant messages with usage.
func estimateFromTranscript(path string) float64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	var total float64
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024) // tolerate long lines
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var tl transcriptLine
		if err := json.Unmarshal(line, &tl); err != nil {
			continue
		}
		if tl.Type != "assistant" || tl.Message.Usage == nil {
			continue
		}
		model := tl.Message.Model
		if model == "" {
			model = "sonnet"
		}
		p := priceFor(model)
		u := tl.Message.Usage
		total += float64(u.InputTokens) * p.In / 1e6
		total += float64(u.OutputTokens) * p.Out / 1e6
		total += float64(u.CacheReadInputTokens) * p.CacheRead / 1e6
		total += float64(u.CacheCreationInputTokens) * p.CacheWrite / 1e6
	}
	return total
}
