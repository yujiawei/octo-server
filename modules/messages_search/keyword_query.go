package messages_search

import (
	"context"
	"strings"

	"github.com/olivere/elastic"
)

// tokenAnalyzer abstracts the IK tokenization step so the keyword-clause
// builders can be unit-tested without standing up an OpenSearch cluster.
// Production wires in an osIKSmartAnalyzer that calls the cluster's
// `_analyze` endpoint; tests provide a stub returning a fixed token slice
// (or an error to exercise the fallback path).
type tokenAnalyzer interface {
	Analyze(ctx context.Context, text string) ([]string, error)
}

// osIKSmartAnalyzer drives OS `_analyze?analyzer=ik_smart`. The endpoint is
// index-independent (we pass no Index()), so it works even when the
// configured read alias is empty/down — the IK plugin lives at the cluster
// level.
type osIKSmartAnalyzer struct {
	client *elastic.Client
}

func newOSIKSmartAnalyzer(c *elastic.Client) osIKSmartAnalyzer {
	return osIKSmartAnalyzer{client: c}
}

// Analyze returns the IK-smart token sequence for `text`, dropping empty
// tokens defensively.
func (a osIKSmartAnalyzer) Analyze(ctx context.Context, text string) ([]string, error) {
	resp, err := a.client.IndexAnalyze().
		Analyzer("ik_smart").
		Text(text).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	tokens := make([]string, 0, len(resp.Tokens))
	for _, t := range resp.Tokens {
		if t.Token == "" {
			continue
		}
		tokens = append(tokens, t.Token)
	}
	return tokens, nil
}

// AnalyzeKeyword runs IK-smart segmentation + stopword filtering against
// `keyword` exactly once, returning the inputs needed by one or more
// buildKeywordClauseFromAnalyzed calls. This lets a handler that issues
// multiple multi_match clauses against the same keyword (e.g. _search_all
// with separate text/file Should branches) reuse one `_analyze` roundtrip.
//
// Return contract per docs/messages-search/2026-06-23-multimatch-or-trap-fix.md §4.1:
//
//   - err != nil  → _analyze failed. effectiveKeyword == keyword, useMSM == true.
//     Feeding this pair to buildKeywordClauseFromAnalyzed yields the §4.4
//     degraded shape (raw keyword + cross_fields + MSM 75%). Caller may
//     warn-log the error; the returned values are still safe to use.
//   - post-strip token slice empty (pure stopwords like "的" / "的了"):
//     effectiveKeyword == keyword (the original, unsplit string), useMSM == false.
//     Builder emits raw keyword + cross_fields, no MSM — preserves the "user
//     literally searches for a function word" semantic.
//   - content tokens remain: effectiveKeyword == content tokens joined with
//     spaces, useMSM == true. Builder emits the joined string + cross_fields +
//     MSM 75%. Stopwords drop out of the MSM denominator so a 5-token query
//     with one stopword becomes a 4-of-4 (75% ≈ 3) test on content words.
func AnalyzeKeyword(ctx context.Context, a tokenAnalyzer, keyword string) (effectiveKeyword string, useMSM bool, err error) {
	tokens, err := a.Analyze(ctx, keyword)
	if err != nil {
		return keyword, true, err
	}
	content := filterStopwords(tokens, defaultStopwords)
	if len(content) == 0 {
		return keyword, false, nil
	}
	return strings.Join(content, " "), true, nil
}

// buildKeywordClauseFromAnalyzed constructs the multi_match clause from a
// previously-computed (effectiveKeyword, useMSM) pair — see AnalyzeKeyword.
// Triggers no _analyze of its own, so a caller can issue multiple
// field-specific clauses (text vs file) from a single analyze.
//
// type=cross_fields is required: under best_fields, MSM is evaluated per
// field, and a token landing in any single field clears the bar, defeating
// the threshold for multi-field corpora (text vs file.name etc).
func buildKeywordClauseFromAnalyzed(effectiveKeyword string, useMSM bool, fields ...string) elastic.Query {
	q := elastic.NewMultiMatchQuery(effectiveKeyword, fields...).Type("cross_fields")
	if useMSM {
		q = q.MinimumShouldMatch("75%")
	}
	return q
}

// buildKeywordClause is the thin wrapper preserved for callers that issue
// exactly one multi_match per keyword (search_messages, search_files). The
// signature, branch semantics, and returned error match the pre-split
// implementation — see AnalyzeKeyword for the branch table.
func buildKeywordClause(ctx context.Context, a tokenAnalyzer, keyword string, fields ...string) (elastic.Query, error) {
	eff, useMSM, err := AnalyzeKeyword(ctx, a, keyword)
	return buildKeywordClauseFromAnalyzed(eff, useMSM, fields...), err
}

// buildKeywordClauseGated is the flag-aware front door used by
// buildSearchMessagesDSL / buildSearchFilesDSL.
//
// When stopwordStripEnabled is false, the `_analyze` call is skipped entirely
// (no IK roundtrip RT) and the result degrades to the §4.4 shape — raw
// keyword + cross_fields + MSM 75%. This is the ops kill switch for the
// stopword-strip pipeline; see SearchConfig.StopwordStripEnabled.
func buildKeywordClauseGated(ctx context.Context, a tokenAnalyzer, stopwordStripEnabled bool, keyword string, fields ...string) (elastic.Query, error) {
	if !stopwordStripEnabled {
		return buildKeywordClauseFromAnalyzed(keyword, true, fields...), nil
	}
	return buildKeywordClause(ctx, a, keyword, fields...)
}
