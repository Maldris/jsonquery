package jsonquery

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

const (
	testJSON = `{
		"name":"John",
		"age":30,
		"motorist": true,
		"cars": [
			{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
			{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
			{ "name":"Fiat", "models":[ "500", "Panda" ] }
		]
	}`
)

func parseString(s string) (*Node, error) {
	return Parse(strings.NewReader(s))
}

func TestParseJsonNumberArray(t *testing.T) {
	s := `[1,2,3,4,5,6]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	// output like below:
	// <element>1</element>
	// <element>2</element>
	// ...
	// <element>6</element>
	if e, g := 6, len(doc.ChildNodes()); e != g {
		t.Fatalf("excepted %v but got %v", e, g)
	}
	var v []string
	for _, n := range doc.ChildNodes() {
		v = append(v, n.InnerText())
	}
	if got, expected := strings.Join(v, ","), "1,2,3,4,5,6"; got != expected {
		t.Fatalf("got %v but expected %v", got, expected)
	}
}

func TestParseJsonObject(t *testing.T) {
	s := `{
		"name":"John",
		"age":31,
		"city":"New York"
	}`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	// output like below:
	// <name>John</name>
	// <age>31</age>
	// <city>New York</city>
	m := make(map[string]string)
	for _, n := range doc.ChildNodes() {
		m[n.Data] = n.InnerText()
	}
	expected := []struct {
		name, value string
	}{
		{"name", "John"},
		{"age", "31"},
		{"city", "New York"},
	}
	for _, v := range expected {
		if e, g := v.value, m[v.name]; e != g {
			t.Fatalf("expected %v=%v,but %v=%v", v.name, e, v.name, g)
		}
	}
}

func TestParseJsonObjectArray(t *testing.T) {
	s := `[
		{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
		{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
        { "name":"Fiat", "models":[ "500", "Panda" ] }
	]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	/**
	<element>
		<name>Ford</name>
		<models>
			<element>Fiesta</element>
			<element>Focus</element>
			<element>Mustang</element>
		</models>
	</element>
	<element>
		<name>BMW</name>
		<models>
			<element>320</element>
			<element>X3</element>
			<element>X5</element>
		</models>
	</element>
	....
	*/
	if e, g := 3, len(doc.ChildNodes()); e != g {
		t.Fatalf("expected %v, but %v", e, g)
	}
	m := make(map[string][]string)
	for _, n := range doc.ChildNodes() {
		// Go to the next of the element list.
		var name string
		var models []string
		for _, e := range n.ChildNodes() {
			if e.Data == "name" {
				// a name node.
				name = e.InnerText()
			} else {
				// a models node.
				for _, k := range e.ChildNodes() {
					models = append(models, k.InnerText())
				}
			}
		}
		// Sort models list.
		sort.Strings(models)
		m[name] = models

	}
	expected := []struct {
		name, value string
	}{
		{"Ford", "Fiesta,Focus,Mustang"},
		{"BMW", "320,X3,X5"},
		{"Fiat", "500,Panda"},
	}
	for _, v := range expected {
		if e, g := v.value, strings.Join(m[v.name], ","); e != g {
			t.Fatalf("expected %v=%v,but %v=%v", v.name, e, v.name, g)
		}
	}
}

func TestParseJson(t *testing.T) {
	doc, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}
	n := doc.SelectElement("name")
	if n == nil {
		t.Fatal("n is nil")
	}
	if n.NextSibling != nil {
		t.Fatal("next sibling should be nil")
	}
	if e, g := "John", n.InnerText(); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	cars := doc.SelectElement("cars")
	if e, g := 3, len(cars.ChildNodes()); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
}

func TestLargeFloat(t *testing.T) {
	s := `{
		"large_number": 365823929453
	}`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	n := doc.SelectElement("large_number")
	if n.InnerText() != "365823929453" {
		t.Fatalf("expected %v but %v", "365823929453", n.InnerText())
	}
}

func TestNodeSelectElement(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	n := top.SelectElement("age")
	if n == nil {
		t.Fatal("n is nil")
	}

	if n.InnerText() != "30" {
		t.Fatalf("expected %v, but %v", "30", n.InnerText())
	}
}

func TestNodeSelectElements(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	ns := top.SelectElements("//name")
	if ns == nil {
		t.Fatal("n is nil")
	}

	if len(ns) != 4 {
		t.Fatalf("len(ns)!=4, got %v", len(ns))
	}

	ns = top.SelectElements("/cars//name")
	if ns == nil {
		t.Fatal("n is nil")
	}

	if len(ns) != 3 {
		t.Fatalf("len(ns)!=3, got %v", len(ns))
	}
}

func TestNodeQuery(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	n, err := top.Query("age")
	if err != nil {
		t.Fatal("error executing query: ", err)
	}
	if n == nil {
		t.Fatal("n is nil")
	}

	if n.InnerText() != "30" {
		t.Fatalf("expected %v, but %v", "30", n.InnerText())
	}
}

func TestNodeQueryAll(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	ns, err := top.QueryAll("//name")
	if err != nil {
		t.Fatal("Error executing query: ", err)
	}
	if ns == nil {
		t.Fatal("n is nil")
	}

	if len(ns) != 4 {
		t.Fatalf("len(ns)!=4, got %v", len(ns))
	}

	ns, err = top.QueryAll("/cars//name")
	if err != nil {
		t.Fatal("Error executing query: ", err)
	}
	if ns == nil {
		t.Fatal("n is nil")
	}

	if len(ns) != 3 {
		t.Fatalf("len(ns)!=3, got %v", len(ns))
	}
}

func TestNodeQuerySelector(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	qa, err := getQuery("age")
	if err != nil {
		t.Fatal(err)
	}

	n := top.QuerySelector(qa)
	if n == nil {
		t.Fatal("n is nil")
	}

	if n.InnerText() != "30" {
		t.Fatalf("expected %v, but %v", "30", n.InnerText())
	}
}

func TestNodeQuerySelectorAll(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	qa, err := getQuery("//name")
	if err != nil {
		t.Fatal(err)
	}

	ns := top.QuerySelectorAll(qa)
	if ns == nil {
		t.Fatal("n is nil")
	}

	if len(ns) != 4 {
		t.Fatalf("len(ns)!=4, got %v", len(ns))
	}

	qa, err = getQuery("/cars//name")
	if err != nil {
		t.Fatal(err)
	}

	ns = top.QuerySelectorAll(qa)
	if ns == nil {
		t.Fatal("n is nil")
	}

	if len(ns) != 3 {
		t.Fatalf("len(ns)!=3, got %v", len(ns))
	}
}

func TestOutputXML(t *testing.T) {
	top, err := parseString(testJSON)
	if err != nil {
		t.Fatal(err)
	}

	xmlContent := top.OutputXML()
	if len(xmlContent) < 21 { // xml version header length
		t.Fatal("xml output less than min length")
	}
	if !strings.Contains(xmlContent, `<?xml version="1.0"?>`) {
		t.Fatal("xml output missing version header")
	}
	if !strings.Contains(xmlContent, `<name>John</name>`) {
		t.Fatal("xml output missing expected content")
	}
	if !strings.Contains(xmlContent, `<age>30</age>`) {
		t.Fatal("xml output missing expected content")
	}
	if !strings.Contains(xmlContent, `<element>Fiesta</element>`) {
		t.Fatal("xml output missing expected content")
	}
}

func TestLoadURLSuccess(t *testing.T) {
	contentTypes := []string{
		"application/json",
		"application/geo+json",
	}

	for _, contentType := range contentTypes {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.Write([]byte(testJSON))
		}))
		defer server.Close()
		_, err := LoadURL(server.URL)
		if err != nil {
			t.Fatal(err)
		}
	}
}
