package messages_search

import (
	"reflect"
	"testing"
)

func TestFilterStopwords_DropsConfigured(t *testing.T) {
	in := []string{"按时", "缴纳", "时间", "的", "女性"}
	got := filterStopwords(in, defaultStopwords)
	want := []string{"按时", "缴纳", "时间", "女性"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFilterStopwords_EmptyInput(t *testing.T) {
	got := filterStopwords(nil, defaultStopwords)
	if got == nil {
		t.Fatalf("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

func TestFilterStopwords_AllStopwords(t *testing.T) {
	in := []string{"的", "了"}
	got := filterStopwords(in, defaultStopwords)
	if len(got) != 0 {
		t.Fatalf("all-stopwords input should yield empty result, got %v", got)
	}
}

func TestFilterStopwords_AllContentWords(t *testing.T) {
	in := []string{"季度", "报告"}
	got := filterStopwords(in, defaultStopwords)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("content-only tokens must pass through unchanged: got %v want %v", got, in)
	}
}

func TestFilterStopwords_DropsEmptyTokens(t *testing.T) {
	in := []string{"", "公司", ""}
	got := filterStopwords(in, defaultStopwords)
	want := []string{"公司"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDefaultStopwords_CoversCommonFunctionWords(t *testing.T) {
	// Regression-pin a few entries that the OR-trap fix specifically targets.
	// If the set is ever trimmed, callers must update this list to keep the
	// trap-fix coverage explicit.
	for _, tok := range []string{"的", "了", "在", "是", "和", "的话"} {
		if _, ok := defaultStopwords[tok]; !ok {
			t.Errorf("defaultStopwords must include %q for the OR-trap fix", tok)
		}
	}
}
