package pkg

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// bm25f parameters
const (
	bm25K1 = 1.2
	bm25B  = 0.25 // low b for uniformly short docs
)

// field attribute weights for multi-field scoring
var fieldWeights = map[string]float64{
	"key":         3.0,
	"label":       2.5,
	"options":     2.0,
	"type":        1.5,
	"description": 1.0,
	"hint":        0.8,
	"example":     0.5,
	"widget":      1.0,
	"section":     0.5,
}

// synonym groups — bidirectional lookup
var synonymGroups = [][]string{
	{"transparency", "opacity", "alpha"},
	{"colour", "color"}, //nolint:misspell // intentional synonym for British English
	{"shortcut", "keybind", "hotkey", "binding"},
	{"font", "typeface"},
	{"bg", "background"},
	{"fg", "foreground"},
	{"cursor", "caret"},
	{"tab", "indent"},
	{"scrollback", "history", "buffer"},
	{"theme", "colorscheme", "palette"},
	{"shell", "command", "program"},
	{"border", "outline", "frame"},
	{"title", "titlebar"},
	{"size", "dimension"},
	{"split", "pane"},
	{"clipboard", "copy", "paste"},
	{"mouse", "pointer"},
	{"bell", "alert", "notification"},
	{"ligature", "liga"},
	{"unfocused", "inactive"},
	{"fullscreen", "maximized"},
	{"env", "environment", "variable"},
	{"padding", "margin", "spacing"},
	{"bold", "weight"},
	{"italic", "slant"},
	{"url", "link", "hyperlink"},
	{"window", "frame"},
	{"path", "file", "directory"},
	{"delay", "timeout", "interval"},
}

var synonymMap map[string][]string

func init() {
	synonymMap = make(map[string][]string)
	for _, group := range synonymGroups {
		for _, term := range group {
			var others []string
			for _, other := range group {
				if other != term {
					others = append(others, other)
				}
			}
			synonymMap[term] = others
		}
	}
}

// ExpandSynonyms returns all synonyms for a term (excluding the term itself).
func ExpandSynonyms(term string) []string {
	return synonymMap[strings.ToLower(term)]
}

// SearchDoc holds precomputed per-field data for bm25f scoring.
// term frequencies are tracked per attribute for proper per-field length normalization.
type SearchDoc struct {
	AttrTF  map[string]map[string]float64 // attribute → term → raw tf
	AttrLen map[string]float64            // attribute → token count
}

// SearchIndex holds corpus-level data for bm25f ranking.
type SearchIndex struct {
	Docs        []SearchDoc
	AttrAvgLen  map[string]float64 // average attribute length across corpus
	DF          map[string]int     // document frequency per term
	N           int                // total number of docs
	Sections    []Section
	sortedTerms []string // sorted DF keys for prefix binary search
	// flat mapping from doc index back to section/field indices
	docSectionIdx []int
	docFieldIdx   []int
}

// SearchResult identifies a matching field with its bm25f score.
type SearchResult struct {
	FieldIdx   int
	SectionIdx int
	Score      float64
	MatchInfo  string // human-readable explanation of the best match
}

// matchKind classifies how a query term was expanded
type matchKind int

const (
	matchExact    matchKind = iota // direct hit or plural variant
	matchSynonym                  // synonym expansion
	matchPrefix                   // prefix expansion
	matchContains                 // substring match
)

// queryTerm pairs a corpus term with a relevance boost
type queryTerm struct {
	term    string
	boost   float64   // 1.0 = exact/plural, 0.5 = prefix/contains/synonym
	kind    matchKind // how this term was derived
	origTok string    // original query token that produced this term
}

func pluralVariant(s string) string {
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s + "s"
}

var stopWords = map[string]struct{}{
	"true": {}, "false": {},
}

func tokenize(s string) []string {
	raw := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := raw[:0]
	for _, tok := range raw {
		if _, stop := stopWords[tok]; !stop {
			out = append(out, tok)
		}
	}
	return out
}

// NewSearchIndex builds a bm25f index from sections.
func NewSearchIndex(sections []Section) *SearchIndex {
	idx := &SearchIndex{
		DF:       make(map[string]int),
		Sections: sections,
	}

	for si, sec := range sections {
		sectionTokens := tokenize(sec.Name)
		for fi := range sec.Fields {
			f := &sec.Fields[fi]
			doc := buildDoc(*f, sectionTokens)
			idx.Docs = append(idx.Docs, doc)
			idx.docSectionIdx = append(idx.docSectionIdx, si)
			idx.docFieldIdx = append(idx.docFieldIdx, fi)

			// document frequency: each unique term counted once per doc
			seen := make(map[string]struct{})
			for _, attrTF := range doc.AttrTF {
				for term := range attrTF {
					if _, ok := seen[term]; !ok {
						idx.DF[term]++
						seen[term] = struct{}{}
					}
				}
			}
		}
	}

	idx.N = len(idx.Docs)

	// compute per-attribute average lengths
	attrTotals := make(map[string]float64)
	for _, doc := range idx.Docs {
		for attr, length := range doc.AttrLen {
			attrTotals[attr] += length
		}
	}
	idx.AttrAvgLen = make(map[string]float64)
	if idx.N > 0 {
		for attr, total := range attrTotals {
			idx.AttrAvgLen[attr] = total / float64(idx.N)
		}
	}

	// build sorted terms for prefix binary search
	idx.sortedTerms = make([]string, 0, len(idx.DF))
	for term := range idx.DF {
		idx.sortedTerms = append(idx.sortedTerms, term)
	}
	sort.Strings(idx.sortedTerms)

	return idx
}

func buildDoc(f Field, sectionTokens []string) SearchDoc {
	doc := SearchDoc{
		AttrTF:  make(map[string]map[string]float64),
		AttrLen: make(map[string]float64),
	}

	addTokens := func(attr, text string) {
		tokens := tokenize(text)
		if len(tokens) == 0 {
			return
		}
		if doc.AttrTF[attr] == nil {
			doc.AttrTF[attr] = make(map[string]float64)
		}
		for _, tok := range tokens {
			doc.AttrTF[attr][tok]++
			doc.AttrLen[attr]++
		}
	}

	addTokens("key", f.Key)
	addTokens("label", f.Label)
	addTokens("type", f.Type)
	addTokens("description", f.Description)
	addTokens("hint", f.Hint)
	addTokens("example", f.Example)
	addTokens("widget", f.Widget)

	for _, opt := range f.Options {
		addTokens("options", opt)
	}

	if len(sectionTokens) > 0 {
		doc.AttrTF["section"] = make(map[string]float64)
		for _, tok := range sectionTokens {
			doc.AttrTF["section"][tok]++
			doc.AttrLen["section"]++
		}
	}

	return doc
}

// Search scores all documents and returns results sorted by score descending.
// multi-word queries use OR semantics with synonym/prefix/contains expansion.
// expanded terms are discounted relative to exact matches.
func (idx *SearchIndex) Search(query string) []SearchResult {
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 || idx.N == 0 {
		return nil
	}

	const expandBoost = 0.5

	var terms []queryTerm
	for _, qt := range queryTokens {
		matched := false
		if _, ok := idx.DF[qt]; ok {
			terms = append(terms, queryTerm{qt, 1.0, matchExact, qt})
			matched = true
		}
		// plural/singular variant at full weight
		if alt := pluralVariant(qt); alt != "" {
			if _, ok := idx.DF[alt]; ok {
				terms = append(terms, queryTerm{alt, 1.0, matchExact, qt})
				matched = true
			}
		}
		if !matched {
			// prefix fallback via binary search on sorted terms
			i := sort.SearchStrings(idx.sortedTerms, qt)
			for ; i < len(idx.sortedTerms) && strings.HasPrefix(idx.sortedTerms[i], qt); i++ {
				terms = append(terms, queryTerm{idx.sortedTerms[i], expandBoost, matchPrefix, qt})
				matched = true
			}
		}
		if !matched {
			// contains fallback
			for _, term := range idx.sortedTerms {
				if strings.Contains(term, qt) {
					terms = append(terms, queryTerm{term, expandBoost, matchContains, qt})
				}
			}
		}
		// synonyms always from original token, discounted
		for _, syn := range ExpandSynonyms(qt) {
			terms = append(terms, queryTerm{syn, expandBoost, matchSynonym, qt})
		}
	}

	var results []SearchResult
	for di := range idx.Docs {
		score := 0.0
		bestContrib := 0.0
		var bestTerm queryTerm
		for _, qt := range terms {
			contrib := qt.boost * idx.bm25f(di, qt.term)
			score += contrib
			if contrib > bestContrib {
				bestContrib = contrib
				bestTerm = qt
			}
		}
		if score > 0 {
			var info string
			switch bestTerm.kind {
			case matchExact:
				info = "match: " + bestTerm.term
			case matchSynonym:
				info = "via " + bestTerm.term + " (synonym)"
			case matchPrefix:
				info = "prefix: " + bestTerm.origTok + "→" + bestTerm.term
			case matchContains:
				info = "contains: " + bestTerm.origTok
			}
			results = append(results, SearchResult{
				FieldIdx:   idx.docFieldIdx[di],
				SectionIdx: idx.docSectionIdx[di],
				Score:      score,
				MatchInfo:  info,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// bm25f computes the bm25f score for a single term in a single document.
// normalizes each attribute's tf against its own average length before combining.
func (idx *SearchIndex) bm25f(docIdx int, term string) float64 {
	doc := idx.Docs[docIdx]

	// per-field weighted tf with per-field length normalization
	combinedTF := 0.0
	for attr, weight := range fieldWeights {
		attrTF := doc.AttrTF[attr]
		if attrTF == nil {
			continue
		}
		tf := attrTF[term]
		if tf == 0 {
			continue
		}
		dl := doc.AttrLen[attr]
		avgdl := idx.AttrAvgLen[attr]
		if avgdl == 0 {
			avgdl = 1
		}
		combinedTF += weight * tf / (1 + bm25B*(dl/avgdl-1))
	}

	if combinedTF <= 0 {
		return 0
	}

	df := idx.DF[term]
	if df == 0 {
		return 0
	}

	idf := math.Log(1 + (float64(idx.N)-float64(df)+0.5)/(float64(df)+0.5))
	return idf * combinedTF * (bm25K1 + 1) / (combinedTF + bm25K1)
}
