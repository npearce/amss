package main

import "strings"

// CuratorReport is the structured output from a KB curation run.
type CuratorReport struct {
	ArticlesReviewed  int              `json:"articles_reviewed"`
	ArticlesUpdated   int              `json:"articles_updated"`
	DuplicatesFlagged []DuplicateEntry `json:"duplicates_flagged"`
	TaggedArticles    []TaggedArticle  `json:"tagged_articles"`
	ScoredArticles    []ScoredArticle  `json:"scored_articles"`
	Summary           string           `json:"summary"`
}

type DuplicateEntry struct {
	ArticleID   string `json:"article_id"`
	DuplicateOf string `json:"duplicate_of"`
	Reason      string `json:"reason"`
}

type TaggedArticle struct {
	ArticleID   string   `json:"article_id"`
	CuratorTags []string `json:"curator_tags"`
}

type ScoredArticle struct {
	ArticleID       string  `json:"article_id"`
	Title           string  `json:"title"`
	UsefulnessScore float64 `json:"usefulness_score"`
	CuratorNotes    string  `json:"curator_notes,omitempty"`
}

// anyContains returns true if msg contains any of the given substrings.
func anyContains(msg string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

func chatResp(kbID, response string) *ChatResponse {
	return &ChatResponse{
		Response:             response,
		KBArticlesReferenced: []string{kbID},
		TicketCreated:        nil,
	}
}

// stubChatResponse pattern-matches keywords in message and returns a plausible
// response with the appropriate KB article reference.
func stubChatResponse(message string) *ChatResponse {
	msg := strings.ToLower(message)

	switch {
	case anyContains(msg, "wcs", "toilet", "flush", "pressure fault"):
		return chatResp("KB-001", "Per KB-001, nominal WCS tank pressure is 14.7 psia ±0.5. "+
			"If pressure drops below range, close the fill valve, check the supply manifold, and notify ECLSS. "+
			"Ground control has been alerted to monitor tank pressure telemetry.")

	case anyContains(msg, "co2", "scrubber", "cartridge"):
		return chatResp("KB-002", "Per KB-002, the CO2 scrubber cartridge must be replaced every 24 hours under nominal crew activity. "+
			"Remove the spent cartridge from Bay 3, insert the replacement from Stowage Locker B4, and verify CO2 partial pressure returns below 0.53 kPa within 30 minutes. "+
			"Notify ECLSS immediately if partial pressure does not stabilize.")

	case anyContains(msg, "comms", "signal", "ka-band", "s-band"):
		return chatResp("KB-003", "Per KB-003, if Ka-band loses lock the system should automatically fall back to S-band within 90 seconds. "+
			"If auto-switchover does not occur, manually select S-band on panel C3 and notify COMM. "+
			"Minimum data rate in S-band emergency mode is 32 kbps.")

	case anyContains(msg, "fire", "smoke"):
		return chatResp("KB-023", "Per KB-023, if smoke or fire is detected, don your PBA immediately and locate the source using the smoke detector panel. "+
			"Activate halon suppression for electrical fires; use the portable fire extinguisher for combustion fires. "+
			"Evacuate the affected module and seal the hatch pending ground control assessment.")

	case anyContains(msg, "radiation", "solar storm", "shelter"):
		return chatResp("KB-014", "Per KB-014, during a solar particle event all crew must shelter in the designated radiation shelter (Node 1 center aisle) within 30 minutes of the CAUTION alert. "+
			"The crew dose limit is 25 mSv per event — monitor the RADAM display continuously. "+
			"Do not exit shelter until GNC confirms the all-clear.")

	case anyContains(msg, "o2", "oxygen", "ogs"):
		return chatResp("KB-015", "Per KB-015, the O2 Generation System (OGS) produces oxygen via water electrolysis at a nominal rate of 0.84 kg/day per crew member. "+
			"If OGS output drops below 80% of nominal, check the water supply pressure and electrolysis cell health indicators. "+
			"Notify ECLSS and switch to emergency O2 reserves if crew O2 partial pressure drops below 19.5 kPa.")

	case anyContains(msg, "solar", "gimbal", "power"):
		return chatResp("KB-004", "Per KB-004, if the solar array gimbal is unresponsive, switch to battery power and attempt a gimbal controller reset via panel P6. "+
			"Do not attempt a manual override without GNC authorization. "+
			"Array output should recover to 85%+ within 10 minutes of a successful reset.")

	case anyContains(msg, "eva", "suit", "pressurization"):
		return chatResp("KB-005", "Per KB-005, all EVA pre-check items must be verified before hatch opening: suit pressure integrity (8.3 psia), O2 quantity (>75%), communications check with CAPCOM, and safety tether attachment. "+
			"Abort the EVA if any pre-check item fails — do not proceed with partial compliance. "+
			"Confirm all items with GC before hatch opening.")

	case anyContains(msg, "exercise", "ared", "cevis"):
		return chatResp("KB-006", "Per KB-006, the crew exercise protocol requires 2.5 hours daily on ARED and CEVIS to maintain bone density and cardiovascular fitness. "+
			"ARED loads should be calibrated per the individualized exercise prescription from the flight surgeon. "+
			"Log session data to the HLTA system after each session.")

	case anyContains(msg, "star tracker", "calibration", "navigation"):
		return chatResp("KB-007", "Per KB-007, star tracker calibration is required after any attitude maneuver exceeding 15 degrees or if pointing error exceeds 0.01 degrees. "+
			"Run the onboard calibration routine from the GNC panel — it takes approximately 8 minutes. "+
			"If calibration fails, switch to backup star tracker B and notify GNC.")

	case anyContains(msg, "thermal", "radiator", "temperature"):
		return chatResp("KB-008", "Per KB-008, if the thermal radiator temperature exceeds nominal range, check the coolant flow rate on the ATCS panel. "+
			"A blockage or pump failure requires switching to the backup radiator loop. "+
			"Notify ECLSS and THERMAL before initiating any bypass.")

	case anyContains(msg, "water", "turbidity", "quality"):
		return chatResp("KB-009", "Per KB-009, water quality testing is required every 72 hours using the in-line sensor package. "+
			"Turbidity above 1 NTU or TDS above 500 ppm indicates a filter replacement is needed. "+
			"Do not consume water from the affected loop until testing confirms return to nominal.")

	default:
		return &ChatResponse{
			Response:             "I searched the KB but couldn't find a specific procedure for that. I'd recommend creating a support ticket so ground control can follow up. Shall I create one?",
			KBArticlesReferenced: []string{},
			TicketCreated:        nil,
		}
	}
}

// stubCuratorReport returns a hardcoded curation report reflecting the known
// state of the seed KB (WCS duplicates KB-018/026/030, untagged articles).
func stubCuratorReport() *CuratorReport {
	return &CuratorReport{
		ArticlesReviewed: 30,
		ArticlesUpdated:  10,
		DuplicatesFlagged: []DuplicateEntry{
			{
				ArticleID:   "KB-018",
				DuplicateOf: "KB-001",
				Reason:      "Both cover WCS flush procedure; KB-001 is more complete with 5 ticket references and a full resolution thread.",
			},
			{
				ArticleID:   "KB-026",
				DuplicateOf: "KB-001",
				Reason:      "WCS pressure fault overlap; KB-001 has the canonical resolution narrative and higher reference count.",
			},
			{
				ArticleID:   "KB-030",
				DuplicateOf: "KB-001",
				Reason:      "WCS operational notes are subsumed by KB-001; stub-level content with no unique procedural detail.",
			},
		},
		TaggedArticles: []TaggedArticle{
			{ArticleID: "KB-004", CuratorTags: []string{"solar-array", "gimbal", "power", "fault-recovery"}},
			{ArticleID: "KB-007", CuratorTags: []string{"star-tracker", "calibration", "gnc", "attitude-control"}},
			{ArticleID: "KB-009", CuratorTags: []string{"water-quality", "turbidity", "filter", "eclss"}},
			{ArticleID: "KB-014", CuratorTags: []string{"radiation", "solar-particle-event", "shelter", "emergency"}},
		},
		ScoredArticles: []ScoredArticle{
			{
				ArticleID:       "KB-001",
				Title:           "WCS Flush Procedure",
				UsefulnessScore: 0.94,
				CuratorNotes:    "High reference count, complete procedure with resolution narrative, actively referenced by resolved tickets.",
			},
			{
				ArticleID:       "KB-005",
				Title:           "EVA Pre-Check Procedure",
				UsefulnessScore: 0.87,
			},
			{
				ArticleID:       "KB-023",
				Title:           "Fire Suppression Protocol",
				UsefulnessScore: 0.82,
			},
			{
				ArticleID:       "KB-018",
				Title:           "WCS Operations Note",
				UsefulnessScore: 0.21,
				CuratorNotes:    "Flagged as duplicate of KB-001. Low reference count and less complete content — recommend consolidation.",
			},
			{
				ArticleID:       "KB-030",
				Title:           "WCS Pressure Monitoring",
				UsefulnessScore: 0.18,
				CuratorNotes:    "Flagged as duplicate of KB-001. Stub article with no procedural detail. Consider archiving.",
			},
		},
		Summary: "Reviewed 30 KB articles. Found 3 WCS duplicate articles (KB-018, KB-026, KB-030) — all flagged as duplicates of the canonical KB-001. " +
			"Added curator tags to 4 articles that had empty tag sets. " +
			"Scored 30 articles for usefulness; 2 articles scored below 0.3 and have been flagged for review or consolidation.",
	}
}
