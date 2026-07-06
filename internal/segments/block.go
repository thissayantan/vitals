package segments

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func init() { Register(&blockSegment{}) }

// blockSegment renders the 5-hour rate-limit usage + reset time,
// e.g. "↻ 5h 65% ↺ 8:10am".
//
// Source: rate_limits.five_hour.{used_percentage,resets_at}.
// Hides when used_percentage is absent or < 0 (the prototype's `// -1` rule).
//
// Options: format ("12h"|"24h"); defaults to 24h when CC_STATUSLINE_24H=1.
type blockSegment struct{}

func (s *blockSegment) Type() string { return "block" }

func (s *blockSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	lim := ctx.Session.RateLimits.FiveHour
	if lim.UsedPercentage == nil {
		return "", false
	}
	pct := int(*lim.UsedPercentage)
	if pct < 0 {
		return "", false
	}

	g := ctx.Theme.Glyphs
	label := "5h"
	if g.BlockIcon != "" {
		label = g.BlockIcon + " 5h"
	}
	out := ctx.Theme.Style("muted").Render(label) + " " +
		styled(ctx, cfg, pctRole(pct), fmt.Sprintf("%d%%", pct))

	if reset := formatReset(lim.ResetsAt, cfg.Options); reset != "" {
		label := reset
		if g.ResetIcon != "" {
			label = g.ResetIcon + " " + reset // space between the reset icon and the time
		}
		out += " " + ctx.Theme.Style("muted").Render(label)
	}
	return out, true
}

// formatReset renders the reset clock time from a unix timestamp, honoring the
// 12h/24h preference. Returns "" when resets_at is unset.
func formatReset(resetsAt int64, opts map[string]any) string {
	if resetsAt <= 0 {
		return ""
	}
	t := time.Unix(resetsAt, 0)

	format := optString(opts, "format", "")
	use24 := format == "24h" || (format == "" && os.Getenv("CC_STATUSLINE_24H") == "1")
	if use24 {
		return t.Format("15:04")
	}
	// 12-hour with lowercase am/pm, no leading zero on hour: "8:10am".
	s := t.Format("3:04PM")
	return strings.ToLower(s)
}
