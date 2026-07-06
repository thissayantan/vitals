package segments

import "fmt"

func init() { Register(&costSegment{}) }

// costSegment renders the session cost: real "$34.13" (API users) or estimated
// teal "≈ $X.XX" (subscription users), prefixed with an estimate glyph. Hides
// when the value is 0.
//
// Options: mode ("auto"|"subscription"|"api", default "auto"). The legacy
// "source" ("auto"|"cc"|"estimate") is still accepted.
type costSegment struct{}

func (s *costSegment) Type() string { return "cost" }

func (s *costSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	mode := optString(cfg.Options, "mode", optString(cfg.Options, "source", "auto"))
	est := ctx.Cost(mode)
	if est.USD <= 0 {
		return "", false
	}

	if est.Estimated {
		amount := fmt.Sprintf("$%.2f", est.USD)
		text := amount
		if g := ctx.Theme.Glyphs.Estimate; g != "" {
			text = g + " " + amount // glyph marks it as an estimate
		} else {
			text = "~" + amount
		}
		return styled(ctx, cfg, "estimate", text), true
	}

	text := fmt.Sprintf("$%.2f", est.USD)
	role := "cost"
	switch {
	case est.USD >= 10:
		role = "crit"
	case est.USD >= 5:
		role = "warn"
	}
	return styled(ctx, cfg, role, text), true
}
