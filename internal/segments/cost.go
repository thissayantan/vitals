package segments

import "fmt"

func init() { Register(&costSegment{}) }

// costSegment renders the session cost: real "$34.13" (API users) or estimated
// teal "~$X.XX" (subscription users, derived from the transcript). Hides when
// the value is 0.
//
// Options: source ("auto"|"cc"|"estimate", default "auto").
type costSegment struct{}

func (s *costSegment) Type() string { return "cost" }

func (s *costSegment) Render(ctx *RenderCtx, cfg SegmentConfig) (string, bool) {
	source := optString(cfg.Options, "source", "auto")
	est := ctx.Cost(source)
	if est.USD <= 0 {
		return "", false
	}

	if est.Estimated {
		text := fmt.Sprintf("~$%.2f", est.USD)
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
