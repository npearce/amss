package main

import (
	"strings"
	"testing"
)

func containsKBRef(resp *ChatResponse, id string) bool {
	for _, ref := range resp.KBArticlesReferenced {
		if ref == id {
			return true
		}
	}
	return false
}

// TestStubChatKeywords verifies that each keyword pattern maps to the correct KB article.
func TestStubChatKeywords(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantKB  string
	}{
		// WCS / toilet (KB-001)
		{"wcs", "wcs flush procedure", "KB-001"},
		{"toilet", "toilet not flushing", "KB-001"},
		{"flush", "how do I flush the system", "KB-001"},
		{"pressure fault", "pressure fault detected on tank", "KB-001"},
		// CO2 / scrubber (KB-002)
		{"co2", "co2 levels elevated", "KB-002"},
		{"scrubber", "scrubber cartridge status", "KB-002"},
		{"cartridge", "replace the cartridge in Bay 3", "KB-002"},
		// Comms (KB-003)
		{"comms", "comms are down", "KB-003"},
		{"signal", "signal lost on downlink", "KB-003"},
		{"ka-band", "ka-band dropout event", "KB-003"},
		{"s-band", "s-band fallback procedure", "KB-003"},
		// Fire / smoke (KB-023)
		{"fire", "fire in the habitation module", "KB-023"},
		{"smoke", "smoke detector alert bay 4", "KB-023"},
		// Radiation / solar storm (KB-014)
		{"radiation", "radiation alert from mission control", "KB-014"},
		{"solar storm", "solar storm warning issued", "KB-014"},
		{"shelter", "shelter in place procedure", "KB-014"},
		// Oxygen / OGS (KB-015)
		{"o2", "o2 partial pressure reading", "KB-015"},
		{"oxygen", "oxygen generation status", "KB-015"},
		{"ogs", "ogs fault code 42", "KB-015"},
		// Solar / gimbal / power (KB-004)
		{"solar", "solar array output low", "KB-004"},
		{"gimbal", "gimbal unresponsive on array 2", "KB-004"},
		{"power", "power drop on bus A", "KB-004"},
		// EVA / suit (KB-005)
		{"eva", "eva pre-check items", "KB-005"},
		{"suit", "suit pressure check", "KB-005"},
		{"pressurization", "suit pressurization nominal", "KB-005"},
		// Exercise (KB-006)
		{"exercise", "daily exercise schedule", "KB-006"},
		{"ared", "ared load calibration", "KB-006"},
		{"cevis", "cevis session data", "KB-006"},
		// Star tracker / navigation (KB-007)
		{"star tracker", "star tracker error after maneuver", "KB-007"},
		{"calibration", "calibration required for gnc", "KB-007"},
		{"navigation", "navigation drift detected", "KB-007"},
		// Thermal (KB-008)
		{"thermal", "thermal control loop fault", "KB-008"},
		{"radiator", "radiator bypass procedure", "KB-008"},
		{"temperature", "temperature spike in node 2", "KB-008"},
		// Water (KB-009)
		{"water", "water quality test results", "KB-009"},
		{"turbidity", "turbidity above limit", "KB-009"},
		{"quality", "water quality check due", "KB-009"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := stubChatResponse(tc.message)
			if !containsKBRef(resp, tc.wantKB) {
				t.Errorf("message %q: want KB ref %q in %v", tc.message, tc.wantKB, resp.KBArticlesReferenced)
			}
			if !strings.Contains(resp.Response, tc.wantKB) {
				t.Errorf("message %q: response does not cite %q:\n%s", tc.message, tc.wantKB, resp.Response)
			}
			if resp.Response == "" {
				t.Errorf("message %q: response is empty", tc.message)
			}
		})
	}
}

// TestStubChatCaseInsensitive verifies keyword matching is case-insensitive.
func TestStubChatCaseInsensitive(t *testing.T) {
	tests := []struct{ msg, wantKB string }{
		{"WCS PRESSURE FAULT", "KB-001"},
		{"CO2 SCRUBBER REPLACEMENT", "KB-002"},
		{"KA-BAND DROPOUT", "KB-003"},
		{"SOLAR ARRAY FAULT", "KB-004"},
	}
	for _, tc := range tests {
		resp := stubChatResponse(tc.msg)
		if !containsKBRef(resp, tc.wantKB) {
			t.Errorf("uppercase %q: want %q, got %v", tc.msg, tc.wantKB, resp.KBArticlesReferenced)
		}
	}
}

// TestStubChatDefault verifies that an unrecognised query returns the fallback response.
func TestStubChatDefault(t *testing.T) {
	resp := stubChatResponse("what is the crew meal rotation schedule")
	if len(resp.KBArticlesReferenced) != 0 {
		t.Errorf("default response should have no KB refs, got %v", resp.KBArticlesReferenced)
	}
	if resp.Response == "" {
		t.Error("default response should not be empty")
	}
	if !strings.Contains(strings.ToLower(resp.Response), "ticket") {
		t.Errorf("default response should suggest creating a ticket, got %q", resp.Response)
	}
}

// TestStubChatDefaultEmptyMessage verifies empty input returns the fallback.
func TestStubChatDefaultEmptyMessage(t *testing.T) {
	resp := stubChatResponse("")
	if len(resp.KBArticlesReferenced) != 0 {
		t.Errorf("empty message should have no KB refs, got %v", resp.KBArticlesReferenced)
	}
}

// TestStubChatResponseNeverNil verifies stubChatResponse always returns a non-nil response.
func TestStubChatResponseNeverNil(t *testing.T) {
	msgs := []string{"", "hello", "wcs", "unknown query xyz"}
	for _, msg := range msgs {
		resp := stubChatResponse(msg)
		if resp == nil {
			t.Errorf("stubChatResponse(%q) returned nil", msg)
		}
		if resp.KBArticlesReferenced == nil {
			t.Errorf("stubChatResponse(%q) KBArticlesReferenced is nil (want empty slice)", msg)
		}
	}
}

// TestStubCuratorReportDuplicates verifies exactly 3 duplicates are flagged.
func TestStubCuratorReportDuplicates(t *testing.T) {
	r := stubCuratorReport()
	if len(r.DuplicatesFlagged) != 3 {
		t.Errorf("want 3 duplicates flagged, got %d", len(r.DuplicatesFlagged))
	}
}

// TestStubCuratorReportWCSDuplicates verifies KB-018/026/030 are all flagged as duplicates of KB-001.
func TestStubCuratorReportWCSDuplicates(t *testing.T) {
	r := stubCuratorReport()
	want := map[string]string{
		"KB-018": "KB-001",
		"KB-026": "KB-001",
		"KB-030": "KB-001",
	}
	for _, d := range r.DuplicatesFlagged {
		expectedDuplicateOf, ok := want[d.ArticleID]
		if !ok {
			t.Errorf("unexpected duplicate entry for %s", d.ArticleID)
			continue
		}
		if d.DuplicateOf != expectedDuplicateOf {
			t.Errorf("%s: duplicate_of = %q, want %q", d.ArticleID, d.DuplicateOf, expectedDuplicateOf)
		}
		if d.Reason == "" {
			t.Errorf("%s: reason should not be empty", d.ArticleID)
		}
		delete(want, d.ArticleID)
	}
	for missing := range want {
		t.Errorf("expected duplicate entry for %s but not found", missing)
	}
}

// TestStubCuratorReportTaggedArticles verifies exactly 4 articles are tagged.
func TestStubCuratorReportTaggedArticles(t *testing.T) {
	r := stubCuratorReport()
	if len(r.TaggedArticles) != 4 {
		t.Errorf("want 4 tagged articles, got %d", len(r.TaggedArticles))
	}
	for _, ta := range r.TaggedArticles {
		if len(ta.CuratorTags) == 0 {
			t.Errorf("article %s has no curator_tags", ta.ArticleID)
		}
		if ta.ArticleID == "" {
			t.Error("tagged article has empty article_id")
		}
	}
}

// TestStubCuratorReportScoredArticles verifies exactly 5 articles are scored.
func TestStubCuratorReportScoredArticles(t *testing.T) {
	r := stubCuratorReport()
	if len(r.ScoredArticles) != 5 {
		t.Errorf("want 5 scored articles, got %d", len(r.ScoredArticles))
	}
}

// TestStubCuratorReportScoreRange verifies all usefulness scores are in [0.0, 1.0].
func TestStubCuratorReportScoreRange(t *testing.T) {
	r := stubCuratorReport()
	for _, sa := range r.ScoredArticles {
		if sa.UsefulnessScore < 0.0 || sa.UsefulnessScore > 1.0 {
			t.Errorf("%s: score %f out of range [0.0, 1.0]", sa.ArticleID, sa.UsefulnessScore)
		}
	}
}

// TestStubCuratorReportLowScoresHaveNotes verifies articles scoring below 0.3 have curator notes.
func TestStubCuratorReportLowScoresHaveNotes(t *testing.T) {
	r := stubCuratorReport()
	for _, sa := range r.ScoredArticles {
		if sa.UsefulnessScore < 0.3 && sa.CuratorNotes == "" {
			t.Errorf("%s: score %f < 0.3 but no curator_notes", sa.ArticleID, sa.UsefulnessScore)
		}
	}
}

// TestStubCuratorReportMixedScores verifies the report contains both high and low scores.
func TestStubCuratorReportMixedScores(t *testing.T) {
	r := stubCuratorReport()
	hasHigh := false
	hasLow := false
	for _, sa := range r.ScoredArticles {
		if sa.UsefulnessScore >= 0.8 {
			hasHigh = true
		}
		if sa.UsefulnessScore < 0.3 {
			hasLow = true
		}
	}
	if !hasHigh {
		t.Error("expected at least one article with score >= 0.8")
	}
	if !hasLow {
		t.Error("expected at least one article with score < 0.3")
	}
}

// TestStubCuratorReportCounts verifies article counts are plausible.
func TestStubCuratorReportCounts(t *testing.T) {
	r := stubCuratorReport()
	if r.ArticlesReviewed <= 0 {
		t.Errorf("articles_reviewed = %d, want > 0", r.ArticlesReviewed)
	}
	if r.ArticlesUpdated <= 0 {
		t.Errorf("articles_updated = %d, want > 0", r.ArticlesUpdated)
	}
	if r.ArticlesUpdated > r.ArticlesReviewed {
		t.Errorf("articles_updated (%d) > articles_reviewed (%d)", r.ArticlesUpdated, r.ArticlesReviewed)
	}
}

// TestStubCuratorReportSummary verifies the summary is non-empty and mentions key findings.
func TestStubCuratorReportSummary(t *testing.T) {
	r := stubCuratorReport()
	if r.Summary == "" {
		t.Error("summary should not be empty")
	}
	for _, want := range []string{"KB-001", "duplicate"} {
		if !strings.Contains(r.Summary, want) {
			t.Errorf("summary should mention %q:\n%s", want, r.Summary)
		}
	}
}
