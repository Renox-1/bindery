package bnf

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestLinkByName(t *testing.T) {
	unlinked := credit{surname: "Carroll", forename: "Lewis", author: true}
	martinA := credit{authorityID: "10000001", surname: "Martin", forename: "Paul", author: true}
	martinB := credit{authorityID: "10000002", surname: "Martin", forename: "Paul", author: true}
	martin := credit{surname: "Martin", forename: "Paul", author: true}
	in := []edition{
		edWith("cb10000001a", "Alice", "", "fre", "", carroll),
		edWith("cb10000002b", "Alice", "", "fre", "", unlinked),
		edWith("cb10000003c", "Un", "", "fre", "", martinA),
		edWith("cb10000004d", "Deux", "", "fre", "", martinB),
		edWith("cb10000005f", "Trois", "", "fre", "", martin),
	}
	out := linkByName(in)
	if got := out[1].credits[0].authorityID; got != "11885918" {
		t.Errorf("unlinked Lewis Carroll = %q, want the link another edition carries", got)
	}
	if got := out[4].credits[0].authorityID; got != "" {
		t.Errorf("unlinked Paul Martin = %q, want none: two people have that name", got)
	}
	if in[1].credits[0].authorityID != "" {
		t.Error("linkByName modified its input")
	}
}

// A page of unlinked authors costs at most maxNameLookups authority lookups,
// so one search stays within the aggregator's per provider budget.
func TestLinkUnlinked_BoundedLookups(t *testing.T) {
	var lookups atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("query"), "aut.accesspoint") {
			lookups.Add(1)
			_, _ = w.Write([]byte(emptySRU))
			return
		}
		var b strings.Builder
		for i, name := range []string{"Un", "Deux", "Trois", "Quatre", "Cinq"} {
			b.WriteString(`<record><recordData><record id="ark:/12148/cb2000000` + string(rune('1'+i)) + `x" type="Bibliographic"><leader>     a    22        450 </leader>` +
				`<datafield tag="200" ind1="1" ind2=" "><subfield code="a">Livre ` + name + `</subfield></datafield>` +
				`<datafield tag="700" ind1=" " ind2=" "><subfield code="a">Auteur` + name + `</subfield><subfield code="b">Jean</subfield></datafield>` +
				`</record></recordData></record>`)
		}
		_, _ = w.Write([]byte(`<searchRetrieveResponse><numberOfRecords>5</numberOfRecords><records>` + b.String() + `</records></searchRetrieveResponse>`))
	}))
	defer srv.Close()
	books, err := testClient(srv.URL).SearchBooks(context.Background(), "livre")
	if err != nil || len(books) != 5 {
		t.Fatalf("books = %d err = %v", len(books), err)
	}
	if got := lookups.Load(); got != maxNameLookups {
		t.Errorf("authority lookups = %d, want the cap of %d", got, maxNameLookups)
	}
}

// An unlinked author is linked through the authority file only when exactly
// one person record has that heading, and a name another result already
// links costs no lookup.
func TestLinkUnlinked(t *testing.T) {
	person := func(id, forename string) string {
		return `<record><recordData><record id="ark:/12148/cb` + id + `x" type="Authority"><leader>     cx  a22        45  </leader>` +
			`<datafield tag="200" ind1=" " ind2="0"><subfield code="a">Martin</subfield><subfield code="b">` + forename + `</subfield></datafield></record></recordData></record>`
	}
	book := func(ark, sub3 string) string {
		return `<record><recordData><record id="ark:/12148/cb` + ark + `x" type="Bibliographic"><leader>     a    22        450 </leader>` +
			`<datafield tag="200" ind1="1" ind2=" "><subfield code="a">Un livre</subfield></datafield>` +
			`<datafield tag="702" ind1=" " ind2=" ">` + sub3 + `<subfield code="a">Martin</subfield><subfield code="b">Paul</subfield><subfield code="c">Auteur du texte</subfield></datafield>` +
			`</record></recordData></record>`
	}
	for _, tc := range []struct {
		name        string
		books       string
		authorities string
		want        string
		lookups     int32
	}{
		{"one person has the heading", book("20000001", ""), person("10000001", "Paul") + person("10000002", "Pauline"), "bnf:author:10000001", 1},
		{"two people share the heading", book("20000001", ""), person("10000001", "Paul") + person("10000002", "Paul"), "", 1},
		{"linked on another edition", book("20000001", "") + book("20000002", `<subfield code="3">10000003</subfield>`), "", "bnf:author:10000003", 0},
	} {
		var lookups atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			records := tc.books
			if strings.Contains(r.URL.Query().Get("query"), "aut.accesspoint") {
				lookups.Add(1)
				records = tc.authorities
			}
			_, _ = w.Write([]byte(`<searchRetrieveResponse><numberOfRecords>2</numberOfRecords><records>` + records + `</records></searchRetrieveResponse>`))
		}))
		books, err := testClient(srv.URL).SearchBooks(context.Background(), "un livre")
		srv.Close()
		if err != nil || len(books) != 1 {
			t.Fatalf("%s: books = %+v err = %v", tc.name, books, err)
		}
		got := ""
		if books[0].Author != nil {
			got = books[0].Author.ForeignID
		}
		if got != tc.want {
			t.Errorf("%s: author = %q, want %q", tc.name, got, tc.want)
		}
		if lookups.Load() != tc.lookups {
			t.Errorf("%s: authority lookups = %d, want %d", tc.name, lookups.Load(), tc.lookups)
		}
	}
}
