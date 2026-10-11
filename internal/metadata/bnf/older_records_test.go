package bnf

import (
	"context"
	"net/url"
	"testing"
)

// Records BnF converted from its older catalogues hold the whole title
// statement in 200 $a, give the original title in brackets or only as a
// uniform title (500), and credit adaptations with relator 100.
func TestOlderRecords(t *testing.T) {
	fs := newFixtureServer(t, func(url.Values) string { return "older_records_alice.xml" })
	page, err := testClient(fs.URL).searchRetrieve(context.Background(), "x", 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	eds := parseRecords(page)
	if len(eds) != 3 {
		t.Fatalf("editions = %d, want 3", len(eds))
	}
	byARK := map[string]edition{}
	for _, ed := range eds {
		byARK[ed.ark] = ed
	}

	statement := byARK["cb32941384z"]
	if statement.title != "Alice au pays des merveilles" {
		t.Errorf("title = %q, want the statement of responsibility cut off", statement.title)
	}
	if statement.originalTitle != "Alice's adventures in wonderland" {
		t.Errorf("original = %q, want the bracketed original", statement.originalTitle)
	}
	if a := primaryAuthor(statement, ""); a == nil || a.displayName() != "Lewis Carroll" {
		t.Errorf("author = %+v", a)
	}

	adaptation := byARK["cb31911075d"]
	if adaptation.originalTitle != "Alice's adventures in Wonderland" {
		t.Errorf("original = %q, want the uniform title", adaptation.originalTitle)
	}
	if a := primaryAuthor(adaptation, ""); a != nil {
		t.Errorf("author = %+v, want none: relator 100 is the author adapted from", a)
	}

	// Relator 330 is a dubious attribution, not the author.
	if a := primaryAuthor(byARK["cb34614049g"], ""); a != nil {
		t.Errorf("author = %+v, want none for relator 330", a)
	}
}

func TestSplitTitleStatement(t *testing.T) {
	cases := []struct{ in, title, original string }{
		{"L'outsider", "L'outsider", ""},
		{`Alice au pays des merveilles ["Alice's adventures in wonderland"]. Traduction de Henriette Rouillard`, "Alice au pays des merveilles", "Alice's adventures in wonderland"},
		{"Alice au pays des merveilles. Traduit par Françoise Durane", "Alice au pays des merveilles", ""},
		{"Alice au pays des merveilles , par Lewis Carroll. Adaptation Gisèle Vallerey", "Alice au pays des merveilles", ""},
		{"Alice au Pays des Merveilles , d'après Lewis Carroll. Par Walt Disney", "Alice au Pays des Merveilles", ""},
		{"Le Petit prince / Antoine de Saint-Exupéry", "Le Petit prince", ""},
		{"Vingt mille lieues sous les mers", "Vingt mille lieues sous les mers", ""},
		{"M. Ripois et la Némésis", "M. Ripois et la Némésis", ""},
	}
	for _, tc := range cases {
		title, original := splitTitleStatement(tc.in)
		if title != tc.title || original != tc.original {
			t.Errorf("splitTitleStatement(%q) = %q, %q; want %q, %q", tc.in, title, original, tc.title, tc.original)
		}
	}
}
