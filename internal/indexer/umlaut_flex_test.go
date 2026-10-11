package indexer

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/vavallee/bindery/internal/indexer/newznab"
)

// umlautFlexCases pin the rule from #3123. A keyword may match the compact
// spelling of a release name (ä to a, ö to o, ü to u) only when the SOURCE
// string held a real ä, ö or ü. A digraph typed in plain ASCII is taken
// literally, because "michael" and "Mueller" look the same once folded and a
// name like Michael has never been an umlaut.
//
// release is matched as a release name would be: folded by NormalizeRelease.
var umlautFlexCases = []struct {
	name    string
	source  string
	release string
	want    bool
}{
	// The bug: a plain English or Hebrew name with ae, oe or ue in it.
	{"michael vs Michal", "Michael", "Michal", false},
	{"samuel vs Samul", "Samuel", "Samul", false},
	{"raphael vs Raphal", "Raphael", "Raphal", false},
	{"joel vs Jol", "Joel", "Jol", false},
	{"noel vs Nol", "Noel", "Nol", false},
	{"manuel vs Manul", "Manuel", "Manul", false},
	{"ruel vs Rul", "Ruel", "Rul", false},
	{"goethe typed ASCII vs Gothe", "Goethe", "Gothe", false},
	// The same names still match themselves.
	{"michael vs Michael", "Michael", "Michael", true},
	{"samuel vs Samuel", "Samuel", "Samuel", true},
	{"raphael vs Raphael", "Raphael", "Raphael", true},
	{"gabriel vs Gabriel", "Gabriel", "Gabriel", true},
	// A real umlaut still meets all three spellings a release can use.
	{"Müller vs Mueller", "Müller", "Mueller", true},
	{"Müller vs Muller", "Müller", "Muller", true},
	{"Müller vs Müller", "Müller", "Müller", true},
	{"Müller vs Miller", "Müller", "Miller", false},
	{"Schönberg vs Schoenberg", "Schönberg", "Schoenberg", true},
	{"Schönberg vs Schonberg", "Schönberg", "Schonberg", true},
	{"Björn vs Bjoern", "Björn", "Bjoern", true},
	{"Björn vs Bjorn", "Björn", "Bjorn", true},
	{"Göthe vs Goethe", "Göthe", "Goethe", true},
	{"Göthe vs Gothe", "Göthe", "Gothe", true},
	{"Geräusch vs Gerausch", "Geräusch", "Gerausch", true},
	{"Gespensterjäger vs Gespensterjaeger", "Gespensterjäger", "Gespensterjaeger", true},
	// Decomposed input is the same letter (#3138): NFD ü is u plus U+0308.
	{"NFD Müller vs Muller", norm.NFD.String("Müller"), "Muller", true},
	{"NFD Müller vs Mueller", norm.NFD.String("Müller"), "Mueller", true},
	{"NFD Björn vs Bjorn", norm.NFD.String("Björn"), "Bjorn", true},
	// A digraph typed in ASCII means exactly that spelling.
	{"Mueller typed vs Mueller", "Mueller", "Mueller", true},
	{"Mueller typed vs Müller", "Mueller", "Müller", true},
	{"Mueller typed vs Muller", "Mueller", "Muller", false},
	{"Schoenberg typed vs Schoenberg", "Schoenberg", "Schoenberg", true},
	{"Schoenberg typed vs Schonberg", "Schoenberg", "Schonberg", false},
	{"Bjoern typed vs Bjorn", "Bjoern", "Bjorn", false},
	{"Bjoern typed vs Bjoern", "Bjoern", "Bjoern", true},
}

// A single word source is a keyword on the title side and a token on the
// author side. Both are asserted, because they are separate producers.
func TestUmlautFlexRequiresARealUmlaut(t *testing.T) {
	for _, tc := range umlautFlexCases {
		t.Run(tc.name, func(t *testing.T) {
			hay := NormalizeRelease(tc.release)

			kws := newznab.SigWords(tc.source)
			if len(kws) != 1 {
				t.Fatalf("SigWords(%q) = %v, want one keyword", tc.source, kws)
			}
			if got := WordBoundaryRegex(kws[0]).MatchString(hay); got != tc.want {
				t.Errorf("title keyword %q (from %q) against release %q (%q): got %v, want %v",
					kws[0], tc.source, tc.release, hay, got, tc.want)
			}
			if got := ContainsPhrase(hay, kws); got != tc.want {
				t.Errorf("phrase %v (from %q) against %q: got %v, want %v", kws, tc.source, hay, got, tc.want)
			}
			if got := containsInOrder(hay, kws); got != tc.want {
				t.Errorf("in order %v (from %q) against %q: got %v, want %v", kws, tc.source, hay, got, tc.want)
			}

			toks := authorTokens(tc.source)
			if len(toks) != 1 {
				t.Fatalf("authorTokens(%q) = %v, want one token", tc.source, toks)
			}
			if got := authorMatchesRelease(hay, toks); got != tc.want {
				t.Errorf("author token %q (from %q) against release %q: got %v, want %v",
					toks[0], tc.source, tc.release, got, tc.want)
			}
		})
	}
}

// TestUmlautFlexEndToEndAuthorGate is the reported symptom: a release by
// Michal must not satisfy a request for an author named Michael. The second
// half pins that a compact umlaut spelling still passes the whole filter,
// including the title identity gate and the different author guard, which
// build their own word lists.
func TestUmlautFlexEndToEndAuthorGate(t *testing.T) {
	got := filterRelevant(toResults(
		"Michael.Crichton.-.Jurassic.Park.epub",
		"Michal.Crichton.-.Jurassic.Park.epub",
	), "Jurassic Park", "Michael Crichton", nil)
	if !contains(got, "Michael.Crichton.-.Jurassic.Park.epub") {
		t.Errorf("the right release was dropped: %v", resultTitles(got))
	}
	if contains(got, "Michal.Crichton.-.Jurassic.Park.epub") {
		t.Errorf("a Michal release was accepted for Michael: %v", resultTitles(got))
	}

	ger := filterRelevant(toResults("Gespensterjager.Cornelia.Funke.epub"),
		"Gespensterjäger", "Cornelia Funke", nil)
	if !contains(ger, "Gespensterjager.Cornelia.Funke.epub") {
		t.Errorf("compact umlaut title was dropped: %v", resultTitles(ger))
	}
	by := filterRelevant(toResults("Die.Stille.ist.ein.Gerausch.-.Herta.Muller.epub"),
		"Die Stille ist ein Geräusch", "Herta Müller", nil)
	if len(by) != 1 {
		t.Errorf("compact title and author both lost their umlaut and must still match: %v", resultTitles(by))
	}
	// A release crediting a different author is still recognised when the
	// requested author's name carries an umlaut, which is the sibling walk in
	// conflictingTitleAuthor comparing words to author tokens by equality.
	other := filterRelevant(toResults("Die Stille ist ein Geraeusch - Hans Schmidt.epub"),
		"Die Stille ist ein Geräusch", "Herta Müller", nil)
	if len(other) != 0 {
		t.Errorf("a different author was accepted: %v", resultTitles(other))
	}
	surname := filterRelevant(toResults("Die Stille ist ein Geraeusch - Mueller.epub"),
		"Die Stille ist ein Geräusch", "Herta Müller", nil)
	if len(surname) != 1 {
		t.Errorf("a surname only credit was treated as a conflicting author: %v", resultTitles(surname))
	}
}

// TestLanguageTagUmlautsStillMatchCompactSpelling: the German tags are written
// in the release alphabet, so they need their umlaut restored for the regex or
// "Ungekurzt" stops being recognised as German (#3123).
func TestLanguageTagUmlautsStillMatchCompactSpelling(t *testing.T) {
	for _, title := range []string{
		"Author - Title Ungekurzt",
		"Author - Title Ungekuerzt",
		"Author - Title Ungekürzt",
		"Author - Title Gekurzt",
		"Author - Title Horbuch",
		"Author - Title Hörbuch",
		"Author - Title Hoerbuch",
	} {
		codes := releaseLanguageCodes(NormalizeRelease(title))
		if len(codes) != 1 || codes[0] != "ger" {
			t.Errorf("%q: language codes = %v, want [ger]", title, codes)
		}
	}
	// A tag whose digraph never was an umlaut is taken literally.
	if codes := releaseLanguageCodes(NormalizeRelease("Author - Title Luisterbok")); len(codes) != 0 {
		t.Errorf("luisterbok is not luisterboek, got %v", codes)
	}
	if codes := releaseLanguageCodes(NormalizeRelease("Author - Title Luisterboek")); len(codes) != 1 || codes[0] != "dut" {
		t.Errorf("luisterboek must still tag dut, got %v", codes)
	}
}

// Every tag whose spelling carries ae, oe or ue must say, here, whether the
// digraph began as an umlaut. A new German marker added without an entry in
// releaseLanguageTagKeywords would silently stop matching its compact form.
func TestLanguageTagDigraphsAreClassified(t *testing.T) {
	// Dutch "boek" and Portuguese "portugues" are not umlauts.
	notUmlauts := map[string]bool{"luisterboek": true, "portugues": true, "portuguese": true}
	for tag := range releaseLanguageTags {
		has := false
		for _, d := range []string{"ae", "oe", "ue"} {
			if strings.Contains(tag, d) {
				has = true
			}
		}
		if !has {
			continue
		}
		_, isUmlaut := releaseLanguageTagKeywords[tag]
		if isUmlaut == notUmlauts[tag] {
			t.Errorf("tag %q carries ae/oe/ue: list it in releaseLanguageTagKeywords if it was an umlaut, or in this test's notUmlauts if not", tag)
		}
	}
}
