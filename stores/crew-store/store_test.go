package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

const seedPath = "seed-data/crew.json"

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(seedPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

// TestNewStore verifies the store initializes from the seed file.
func TestNewStore(t *testing.T) {
	s := newTestStore(t)
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

// TestNewStoreInvalidPath fails on a bad seed path.
func TestNewStoreInvalidPath(t *testing.T) {
	_, err := NewStore("/nonexistent/path/crew.json")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

// TestLoadSeedCrewCount expects exactly 20 crew members.
func TestLoadSeedCrewCount(t *testing.T) {
	s := newTestStore(t)
	if got := len(s.crew); got != 20 {
		t.Errorf("crew count = %d, want 20", got)
	}
}

// TestLoadSeedSpecificMember verifies known fields for wiseman-r.
func TestLoadSeedSpecificMember(t *testing.T) {
	s := newTestStore(t)
	m, ok := s.crew["wiseman-r"]
	if !ok {
		t.Fatal("wiseman-r not found")
	}
	if m.Name != "Reid Wiseman" {
		t.Errorf("name = %q, want %q", m.Name, "Reid Wiseman")
	}
	if m.Mission != "artemis-ii" {
		t.Errorf("mission = %q, want artemis-ii", m.Mission)
	}
	if m.Role != "Commander" {
		t.Errorf("role = %q, want Commander", m.Role)
	}
	if m.Status != "active" {
		t.Errorf("status = %q, want active", m.Status)
	}
	if m.Persona != "astronaut" {
		t.Errorf("persona = %q, want astronaut", m.Persona)
	}
}

// TestLoadSeedGroundControl verifies ground-control members have mission="all".
func TestLoadSeedGroundControl(t *testing.T) {
	s := newTestStore(t)
	gcIDs := []string{"gc-flight", "gc-capcom", "gc-eclss", "gc-gnc", "gc-systems", "gc-surgeon", "gc-comm", "gc-eva"}
	for _, id := range gcIDs {
		m, ok := s.crew[id]
		if !ok {
			t.Errorf("ground control member %s not found", id)
			continue
		}
		if m.Mission != "all" {
			t.Errorf("%s mission = %q, want all", id, m.Mission)
		}
		if m.Persona != "ground-control" {
			t.Errorf("%s persona = %q, want ground-control", id, m.Persona)
		}
	}
}

// TestLoadSeedNextConvID starts at 1 when there are no seed conversations.
func TestLoadSeedNextConvID(t *testing.T) {
	s := newTestStore(t)
	if s.nextConvID != 1 {
		t.Errorf("nextConvID = %d, want 1", s.nextConvID)
	}
}

// TestLoadSeedEmptyConversations verifies no conversations are loaded from seed.
func TestLoadSeedEmptyConversations(t *testing.T) {
	s := newTestStore(t)
	if got := len(s.conversations); got != 0 {
		t.Errorf("conversation count = %d, want 0", got)
	}
}

// TestGetCrewMember returns the correct member for a known ID.
func TestGetCrewMember(t *testing.T) {
	s := newTestStore(t)
	m := s.GetCrewMember("gc-flight")
	if m == nil {
		t.Fatal("expected non-nil member")
	}
	if m.Name != "Rachel Torres" {
		t.Errorf("name = %q, want Rachel Torres", m.Name)
	}
}

// TestGetCrewMemberNotFound returns nil for unknown IDs.
func TestGetCrewMemberNotFound(t *testing.T) {
	s := newTestStore(t)
	if m := s.GetCrewMember("nobody"); m != nil {
		t.Errorf("expected nil, got %+v", m)
	}
}

// TestListCrewNoFilters returns all 20 members.
func TestListCrewNoFilters(t *testing.T) {
	s := newTestStore(t)
	members, total := s.ListCrew("", "", 100, 0)
	if total != 20 {
		t.Errorf("total = %d, want 20", total)
	}
	if len(members) != 20 {
		t.Errorf("len = %d, want 20", len(members))
	}
}

// TestListCrewSortedByID verifies alphabetical sort when no search.
func TestListCrewSortedByID(t *testing.T) {
	s := newTestStore(t)
	members, _ := s.ListCrew("", "", 100, 0)
	for i := 1; i < len(members); i++ {
		if members[i-1].ID > members[i].ID {
			t.Errorf("not sorted: %s > %s", members[i-1].ID, members[i].ID)
		}
	}
}

// TestListCrewFilterArtemisII returns 4 astronauts + 8 ground control = 12.
func TestListCrewFilterArtemisII(t *testing.T) {
	s := newTestStore(t)
	members, total := s.ListCrew("artemis-ii", "", 100, 0)
	if total != 12 {
		t.Errorf("total = %d, want 12", total)
	}
	if len(members) != 12 {
		t.Errorf("len = %d, want 12", len(members))
	}
}

// TestListCrewFilterArtemisIIIncludesGC confirms ground-control members are included.
func TestListCrewFilterArtemisIIIncludesGC(t *testing.T) {
	s := newTestStore(t)
	members, _ := s.ListCrew("artemis-ii", "", 100, 0)
	gcCount := 0
	for _, m := range members {
		if m.Mission == "all" {
			gcCount++
		}
	}
	if gcCount != 8 {
		t.Errorf("gc count = %d, want 8", gcCount)
	}
}

// TestListCrewFilterArtemisIII returns 4 astronauts + 8 ground control = 12.
func TestListCrewFilterArtemisIII(t *testing.T) {
	s := newTestStore(t)
	_, total := s.ListCrew("artemis-iii", "", 100, 0)
	if total != 12 {
		t.Errorf("total = %d, want 12", total)
	}
}

// TestListCrewFilterArtemisIV returns 4 astronauts + 8 ground control = 12.
func TestListCrewFilterArtemisIV(t *testing.T) {
	s := newTestStore(t)
	_, total := s.ListCrew("artemis-iv", "", 100, 0)
	if total != 12 {
		t.Errorf("total = %d, want 12", total)
	}
}

// TestListCrewFilterAll returns only ground-control (mission=="all") members = 8.
func TestListCrewFilterAll(t *testing.T) {
	s := newTestStore(t)
	members, total := s.ListCrew("all", "", 100, 0)
	if total != 8 {
		t.Errorf("total = %d, want 8", total)
	}
	for _, m := range members {
		if m.Mission != "all" {
			t.Errorf("member %s has mission %q, want all", m.ID, m.Mission)
		}
	}
}

// TestListCrewSearch finds members by name token.
func TestListCrewSearch(t *testing.T) {
	s := newTestStore(t)
	// "wiseman" appears in the name "Reid Wiseman"
	members, total := s.ListCrew("", "wiseman", 100, 0)
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(members) == 0 || members[0].ID != "wiseman-r" {
		t.Errorf("expected wiseman-r, got %+v", members)
	}
}

// TestListCrewSearchRole finds by role token.
func TestListCrewSearchRole(t *testing.T) {
	s := newTestStore(t)
	// "Commander" appears in 3 crew members (artemis II, III, IV commanders)
	_, total := s.ListCrew("", "commander", 100, 0)
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

// TestListCrewSearchNoMatch returns empty for unknown terms.
func TestListCrewSearchNoMatch(t *testing.T) {
	s := newTestStore(t)
	members, total := s.ListCrew("", "xyznonexistent", 100, 0)
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(members) != 0 {
		t.Errorf("len = %d, want 0", len(members))
	}
}

// TestListCrewPagination verifies limit and offset.
func TestListCrewPagination(t *testing.T) {
	s := newTestStore(t)
	page1, _ := s.ListCrew("", "", 5, 0)
	page2, _ := s.ListCrew("", "", 5, 5)
	if len(page1) != 5 {
		t.Errorf("page1 len = %d, want 5", len(page1))
	}
	if len(page2) != 5 {
		t.Errorf("page2 len = %d, want 5", len(page2))
	}
	seen := make(map[string]bool)
	for _, m := range page1 {
		seen[m.ID] = true
	}
	for _, m := range page2 {
		if seen[m.ID] {
			t.Errorf("member %s appeared in both pages", m.ID)
		}
	}
}

// TestListCrewOffsetBeyondTotal returns empty slice with correct total.
func TestListCrewOffsetBeyondTotal(t *testing.T) {
	s := newTestStore(t)
	members, total := s.ListCrew("", "", 10, 999)
	if total != 20 {
		t.Errorf("total = %d, want 20", total)
	}
	if len(members) != 0 {
		t.Errorf("len = %d, want 0", len(members))
	}
}

// TestCreateConversation creates a conversation with all required fields.
func TestCreateConversation(t *testing.T) {
	s := newTestStore(t)
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "session-1", "How do I fix the ECLSS?", "Check the filters.", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil conversation")
	}
	if c.CrewID != "wiseman-r" {
		t.Errorf("crew_id = %q, want wiseman-r", c.CrewID)
	}
	if c.Mission != "artemis-ii" {
		t.Errorf("mission = %q, want artemis-ii", c.Mission)
	}
	if c.Query != "How do I fix the ECLSS?" {
		t.Errorf("query = %q", c.Query)
	}
}

// TestCreateConversationIDFormat verifies conv-NNNN format.
func TestCreateConversationIDFormat(t *testing.T) {
	s := newTestStore(t)
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if c.ID != "conv-0001" {
		t.Errorf("id = %q, want conv-0001", c.ID)
	}
}

// TestCreateConversationSequentialIDs verifies IDs increment.
func TestCreateConversationSequentialIDs(t *testing.T) {
	s := newTestStore(t)
	c1, _ := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q1", "r1", nil, nil)
	c2, _ := s.CreateConversation("glover-v", "artemis-ii", "s1", "q2", "r2", nil, nil)
	if c1.ID != "conv-0001" {
		t.Errorf("first id = %q, want conv-0001", c1.ID)
	}
	if c2.ID != "conv-0002" {
		t.Errorf("second id = %q, want conv-0002", c2.ID)
	}
}

// TestCreateConversationInvalidCrewID returns error for unknown crew_id.
func TestCreateConversationInvalidCrewID(t *testing.T) {
	s := newTestStore(t)
	_, err := s.CreateConversation("nobody", "artemis-ii", "s1", "q", "r", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid crew_id")
	}
	if !strings.Contains(err.Error(), "nobody") {
		t.Errorf("error should mention crew_id: %v", err)
	}
}

// TestCreateConversationDefaultKBArticles nil becomes empty slice.
func TestCreateConversationDefaultKBArticles(t *testing.T) {
	s := newTestStore(t)
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if c.KBArticlesReferenced == nil {
		t.Error("KBArticlesReferenced should not be nil")
	}
	if len(c.KBArticlesReferenced) != 0 {
		t.Errorf("KBArticlesReferenced len = %d, want 0", len(c.KBArticlesReferenced))
	}
}

// TestCreateConversationTimestampRFC3339 verifies the timestamp is valid RFC3339.
func TestCreateConversationTimestampRFC3339(t *testing.T) {
	s := newTestStore(t)
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, c.Timestamp); err != nil {
		t.Errorf("timestamp %q is not valid RFC3339: %v", c.Timestamp, err)
	}
}

// TestCreateConversationWithTicket sets the ticket_created field.
func TestCreateConversationWithTicket(t *testing.T) {
	s := newTestStore(t)
	ticket := "AMSS-042"
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, &ticket)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if c.TicketCreated == nil {
		t.Fatal("expected ticket_created to be set")
	}
	if *c.TicketCreated != "AMSS-042" {
		t.Errorf("ticket_created = %q, want AMSS-042", *c.TicketCreated)
	}
}

// TestCreateConversationWithKBArticles stores referenced KB articles.
func TestCreateConversationWithKBArticles(t *testing.T) {
	s := newTestStore(t)
	articles := []string{"KB-001", "KB-002"}
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", articles, nil)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if len(c.KBArticlesReferenced) != 2 {
		t.Errorf("kb_articles len = %d, want 2", len(c.KBArticlesReferenced))
	}
}

// TestGetCrewActivityEmpty returns empty list for a crew member with no conversations.
func TestGetCrewActivityEmpty(t *testing.T) {
	s := newTestStore(t)
	convs, total, err := s.GetCrewActivity("wiseman-r", 20)
	if err != nil {
		t.Fatalf("GetCrewActivity: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(convs) != 0 {
		t.Errorf("len = %d, want 0", len(convs))
	}
}

// TestGetCrewActivityReturnsConversations returns conversations for a crew member.
func TestGetCrewActivityReturnsConversations(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q1", "r1", nil, nil)
	s.CreateConversation("wiseman-r", "artemis-ii", "s2", "q2", "r2", nil, nil)
	s.CreateConversation("glover-v", "artemis-ii", "s3", "q3", "r3", nil, nil)

	convs, total, err := s.GetCrewActivity("wiseman-r", 20)
	if err != nil {
		t.Fatalf("GetCrewActivity: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(convs) != 2 {
		t.Errorf("len = %d, want 2", len(convs))
	}
	for _, c := range convs {
		if c.CrewID != "wiseman-r" {
			t.Errorf("got conversation for crew_id %q, want wiseman-r", c.CrewID)
		}
	}
}

// TestGetCrewActivityNewestFirst verifies descending timestamp order.
func TestGetCrewActivityNewestFirst(t *testing.T) {
	s := newTestStore(t)
	c1, _ := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q1", "r1", nil, nil)
	c2, _ := s.CreateConversation("wiseman-r", "artemis-ii", "s2", "q2", "r2", nil, nil)

	// Force distinct timestamps so ordering is deterministic.
	s.conversations[c1.ID].Timestamp = "2024-01-01T10:00:00Z"
	s.conversations[c2.ID].Timestamp = "2024-01-01T12:00:00Z"

	convs, _, _ := s.GetCrewActivity("wiseman-r", 20)
	if len(convs) < 2 {
		t.Fatal("expected at least 2 conversations")
	}
	if convs[0].ID != c2.ID {
		t.Errorf("first conv = %q, want %q (newest)", convs[0].ID, c2.ID)
	}
}

// TestGetCrewActivityLimit applies the limit.
func TestGetCrewActivityLimit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 5; i++ {
		s.CreateConversation("wiseman-r", "artemis-ii", fmt.Sprintf("s%d", i), "q", "r", nil, nil)
	}
	convs, total, _ := s.GetCrewActivity("wiseman-r", 3)
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(convs) != 3 {
		t.Errorf("len = %d, want 3", len(convs))
	}
}

// TestGetCrewActivityNotFound returns an error for an unknown crew ID.
func TestGetCrewActivityNotFound(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.GetCrewActivity("nobody", 20)
	if err == nil {
		t.Fatal("expected error for unknown crew_id")
	}
}

// TestListConversationsNoFilter returns all conversations.
func TestListConversationsNoFilter(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q1", "r1", nil, nil)
	s.CreateConversation("glover-v", "artemis-ii", "s2", "q2", "r2", nil, nil)
	convs, total := s.ListConversations(map[string]string{}, 50, 0)
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(convs) != 2 {
		t.Errorf("len = %d, want 2", len(convs))
	}
}

// TestListConversationsFilterByCrewID filters correctly.
func TestListConversationsFilterByCrewID(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q1", "r1", nil, nil)
	s.CreateConversation("glover-v", "artemis-ii", "s2", "q2", "r2", nil, nil)
	s.CreateConversation("wiseman-r", "artemis-ii", "s3", "q3", "r3", nil, nil)
	convs, total := s.ListConversations(map[string]string{"crew_id": "wiseman-r"}, 50, 0)
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	for _, c := range convs {
		if c.CrewID != "wiseman-r" {
			t.Errorf("unexpected crew_id %q", c.CrewID)
		}
	}
}

// TestListConversationsFilterByMission filters by mission.
func TestListConversationsFilterByMission(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q1", "r1", nil, nil)
	s.CreateConversation("patel-a", "artemis-iii", "s2", "q2", "r2", nil, nil)
	convs, total := s.ListConversations(map[string]string{"mission": "artemis-iii"}, 50, 0)
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(convs) > 0 && convs[0].Mission != "artemis-iii" {
		t.Errorf("mission = %q, want artemis-iii", convs[0].Mission)
	}
}

// TestListConversationsFilterBySessionID filters by session_id.
func TestListConversationsFilterBySessionID(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "session-A", "q1", "r1", nil, nil)
	s.CreateConversation("wiseman-r", "artemis-ii", "session-B", "q2", "r2", nil, nil)
	convs, total := s.ListConversations(map[string]string{"session_id": "session-A"}, 50, 0)
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(convs) > 0 && convs[0].SessionID != "session-A" {
		t.Errorf("session_id = %q", convs[0].SessionID)
	}
}

// TestListConversationsPagination verifies limit and offset.
func TestListConversationsPagination(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 6; i++ {
		s.CreateConversation("wiseman-r", "artemis-ii", fmt.Sprintf("s%d", i), "q", "r", nil, nil)
	}
	page1, _ := s.ListConversations(map[string]string{}, 3, 0)
	page2, _ := s.ListConversations(map[string]string{}, 3, 3)
	if len(page1) != 3 {
		t.Errorf("page1 len = %d, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("page2 len = %d, want 3", len(page2))
	}
	seen := make(map[string]bool)
	for _, c := range page1 {
		seen[c.ID] = true
	}
	for _, c := range page2 {
		if seen[c.ID] {
			t.Errorf("conv %s appeared in both pages", c.ID)
		}
	}
}

// TestListConversationsOffsetBeyondTotal returns empty with correct total.
func TestListConversationsOffsetBeyondTotal(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	convs, total := s.ListConversations(map[string]string{}, 10, 999)
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(convs) != 0 {
		t.Errorf("len = %d, want 0", len(convs))
	}
}

// TestResetRestoresCrewCount returns 20 crew members after reset.
func TestResetRestoresCrewCount(t *testing.T) {
	s := newTestStore(t)
	count, err := s.Reset()
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if count != 20 {
		t.Errorf("count = %d, want 20", count)
	}
}

// TestResetClearsConversations conversations are gone after reset.
func TestResetClearsConversations(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	s.CreateConversation("glover-v", "artemis-ii", "s2", "q", "r", nil, nil)
	if len(s.conversations) != 2 {
		t.Fatalf("pre-reset: expected 2 conversations, got %d", len(s.conversations))
	}
	s.Reset()
	if len(s.conversations) != 0 {
		t.Errorf("post-reset: expected 0 conversations, got %d", len(s.conversations))
	}
}

// TestResetResetsNextConvID resets nextConvID to 1.
func TestResetResetsNextConvID(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	s.CreateConversation("wiseman-r", "artemis-ii", "s2", "q", "r", nil, nil)
	s.Reset()
	if s.nextConvID != 1 {
		t.Errorf("nextConvID after reset = %d, want 1", s.nextConvID)
	}
}

// TestResetRestoresCrew verifies crew members are intact after reset.
func TestResetRestoresCrew(t *testing.T) {
	s := newTestStore(t)
	s.Reset()
	m := s.GetCrewMember("wiseman-r")
	if m == nil {
		t.Error("wiseman-r missing after reset")
	}
}

// TestResetNewIDsStartFromOne verifies new conversations after reset get conv-0001.
func TestResetNewIDsStartFromOne(t *testing.T) {
	s := newTestStore(t)
	s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	s.Reset()
	c, err := s.CreateConversation("wiseman-r", "artemis-ii", "s1", "q", "r", nil, nil)
	if err != nil {
		t.Fatalf("CreateConversation after reset: %v", err)
	}
	if c.ID != "conv-0001" {
		t.Errorf("id after reset = %q, want conv-0001", c.ID)
	}
}

// TestThreadSafety runs concurrent reads and creates without data races.
func TestThreadSafety(t *testing.T) {
	s := newTestStore(t)

	var wg sync.WaitGroup

	// concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.ListCrew("artemis-ii", "", 50, 0)
			s.GetCrewMember("wiseman-r")
			s.ListConversations(map[string]string{}, 50, 0)
		}()
	}

	// concurrent creates
	crewIDs := []string{"wiseman-r", "glover-v", "koch-c", "hansen-j", "gc-flight"}
	for i, id := range crewIDs {
		wg.Add(1)
		go func(idx int, crewID string) {
			defer wg.Done()
			s.CreateConversation(crewID, "artemis-ii", fmt.Sprintf("s%d", idx), "q", "r", nil, nil)
		}(i, id)
	}

	wg.Wait()
}
