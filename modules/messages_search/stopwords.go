package messages_search

// defaultStopwords is the Chinese stopword set used by buildKeywordClause to
// strip high-frequency function words (虚词) from an analyzed token stream
// before constructing the multi_match clause.
//
// See docs/messages-search/2026-06-23-multimatch-or-trap-fix.md §4.3 — the
// set is conservative and hardcoded for the first cut; once telemetry shows
// which other tokens still drive OR-trap noise we can promote this to config.
var defaultStopwords = map[string]struct{}{
	"的": {}, "了": {}, "在": {}, "是": {}, "和": {}, "也": {},
	"就": {}, "都": {}, "而": {}, "及": {}, "与": {}, "或": {},
	"把": {}, "被": {}, "对": {}, "向": {}, "从": {}, "到": {},
	"给": {}, "让": {}, "比": {}, "为": {}, "以": {}, "于": {}, "由": {},
	"的话": {}, "之类": {}, "之中": {},
}

// filterStopwords returns the tokens not present in `set`, preserving the
// original order. Returns a non-nil empty slice when every input token is a
// stopword (callers branch on len == 0 to fall back to the original keyword).
func filterStopwords(tokens []string, set map[string]struct{}) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if _, drop := set[t]; drop {
			continue
		}
		out = append(out, t)
	}
	return out
}
