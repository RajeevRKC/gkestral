package gemini

// EnableSearchGrounding adds the google_search tool to a
// GenerateContentRequest, enabling Google Search grounding. The model will
// access Google Search when it determines that grounding would improve the
// response quality.
//
// For Gemini 2.x models, pass nil for dynamicRetrievalConfig.
// For Gemini 3.x models combined with function calling, use
// EnableSearchGroundingValidated which sets the VALIDATED mode required for
// the search + function-calling combination.
func EnableSearchGrounding(request *GenerateContentRequest) {
	addGoogleSearchTool(request, nil)
}

// EnableSearchGroundingValidated adds search grounding with
// DynamicRetrievalConfig mode=VALIDATED, required when combining search
// grounding with function calling on Gemini 3.x models. The dynamicThreshold
// controls the confidence level (0.0-1.0) above which the model uses search
// results. A nil threshold uses the API default.
func EnableSearchGroundingValidated(request *GenerateContentRequest, dynamicThreshold *float64) {
	cfg := &DynamicRetrievalConfig{
		Mode: "VALIDATED",
	}
	if dynamicThreshold != nil {
		cfg.DynamicThreshold = dynamicThreshold
	}
	addGoogleSearchTool(request, cfg)
}

// addGoogleSearchTool appends a google_search tool entry to the request. If
// a google_search tool already exists it is updated rather than duplicated.
func addGoogleSearchTool(request *GenerateContentRequest, cfg *DynamicRetrievalConfig) {
	gs := &GoogleSearch{}
	if cfg != nil {
		gs.DynamicRetrievalConfig = cfg
	}

	// Check if a google_search tool already exists.
	for i := range request.Tools {
		if request.Tools[i].GoogleSearch != nil {
			request.Tools[i].GoogleSearch = gs
			return
		}
	}

	// Append a new tool entry.
	request.Tools = append(request.Tools, Tool{
		GoogleSearch: gs,
	})
}

// HasSearchGrounding reports whether the request has search grounding enabled.
func HasSearchGrounding(request *GenerateContentRequest) bool {
	for _, t := range request.Tools {
		if t.GoogleSearch != nil {
			return true
		}
	}
	return false
}

// ExtractGroundingMetadata returns the grounding metadata from the first
// candidate of a Gemini response, or nil if no grounding data is present.
func ExtractGroundingMetadata(resp *GenerateContentResponse) *GroundingMetadata {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}
	return resp.Candidates[0].GroundingMetadata
}

// ExtractCitationsFromResponse is a convenience wrapper that extracts
// structured Citation objects directly from a response. It combines
// ExtractGroundingMetadata and ExtractCitationsFromMetadata.
func ExtractCitationsFromResponse(resp *GenerateContentResponse) []Citation {
	gm := ExtractGroundingMetadata(resp)
	return ExtractCitationsFromMetadata(gm)
}

// ExtractCitationsFromMetadata converts GroundingMetadata into structured
// Citation objects with URL, title, confidence scores, and supported text.
// This is a richer version than the streaming.go extractCitations helper --
// it cross-references GroundingSupports with GroundingChunks to produce
// fully-populated Citation objects.
func ExtractCitationsFromMetadata(gm *GroundingMetadata) []Citation {
	if gm == nil || len(gm.GroundingChunks) == 0 {
		return nil
	}

	// If there are no supports, fall back to basic chunk-only citations.
	if len(gm.GroundingSupports) == 0 {
		return basicCitationsFromChunks(gm.GroundingChunks)
	}

	// Build rich citations by cross-referencing supports with chunks.
	var citations []Citation
	for _, support := range gm.GroundingSupports {
		text := ""
		if support.Segment != nil {
			text = support.Segment.Text
		}

		for i, chunkIdx := range support.GroundingChunkIndices {
			if chunkIdx < 0 || chunkIdx >= len(gm.GroundingChunks) {
				continue
			}
			chunk := gm.GroundingChunks[chunkIdx]
			if chunk.Web == nil {
				continue
			}

			confidence := 0.0
			if i < len(support.ConfidenceScores) {
				confidence = support.ConfidenceScores[i]
			}

			citations = append(citations, Citation{
				URL:           chunk.Web.URI,
				Title:         chunk.Web.Title,
				Confidence:    confidence,
				SupportedText: text,
			})
		}
	}
	return citations
}

// basicCitationsFromChunks creates Citation objects from chunks when no
// GroundingSupports are available (no confidence or text mapping).
func basicCitationsFromChunks(chunks []GroundingChunk) []Citation {
	var citations []Citation
	for _, chunk := range chunks {
		if chunk.Web != nil {
			citations = append(citations, Citation{
				URL:   chunk.Web.URI,
				Title: chunk.Web.Title,
			})
		}
	}
	return citations
}

// SearchQueriesFromMetadata extracts the web search queries the model used
// for grounding. Useful for debugging and displaying search provenance.
func SearchQueriesFromMetadata(gm *GroundingMetadata) []string {
	if gm == nil {
		return nil
	}
	return gm.WebSearchQueries
}

// EnableSearchWithFunctionCalling configures a request for the Gemini 3.x
// combination of search grounding and function calling. This requires:
// 1. google_search tool with DynamicRetrievalConfig mode=VALIDATED
// 2. Function declarations in a separate Tool entry
// 3. The model must be a 3.x variant
//
// The function sets up the search tool; callers must also add function
// declarations separately via ApplyTools.
func EnableSearchWithFunctionCalling(request *GenerateContentRequest, dynamicThreshold *float64) {
	EnableSearchGroundingValidated(request, dynamicThreshold)
}
