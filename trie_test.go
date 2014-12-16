package route

import (
	"log"
	"testing"
)

func TestPathSplit(t *testing.T) {

	parts, err := Split("/before/:id/middle/:property.*format")
	if err != nil {
		log.Fatal(err)
	}
	if parts["before"].Parameter.Name == "id" {

	}
}

func TestPathInsert(t *testing.T) {

	trie := New()
	if trie.root == nil {
		t.Fatal()
	}

	trie.AddRoute("GET", "/", "1")
	if trie.root.Children["/"] == nil {
		t.Fatal()
	}

	trie.AddRoute("GET", "/r", "2")
	if trie.root.Children["/"].Children["r"] == nil {
		t.Fatal()
	}

	trie.AddRoute("GET", "/r/", "3")
	if trie.root.Children["/"].Children["r"].Children["/"] == nil {
		t.Fatal()
	}
}

func TestTrieCompression(t *testing.T) {

	trie := New()
	trie.AddRoute("GET", "/abc", "3")
	trie.AddRoute("GET", "/adc", "3")

	// before compression
	if trie.root.Children["/"].Children["a"].Children["b"].Children["c"] == nil {
		t.Fatal()
	}
	if trie.root.Children["/"].Children["a"].Children["d"].Children["c"] == nil {
		t.Fatal()
	}

	trie.Compress()

	// after compression
	if trie.root.Children["/abc"] == nil {
		t.Fatalf("%+v", trie.root)
	}
	if trie.root.Children["/adc"] == nil {
		t.Fatalf("%+v", trie.root)
	}
}

func TestParamInsert(t *testing.T) {
	trie := New()

	trie.AddRoute("GET", "/:id/", "")
	if trie.root.Children["/"].ParamChild.Children["/"] == nil {
		t.Fatal()
	}

	trie.printDebug()

	if trie.root.Children["/"].ParamName != "id" {
		t.Fatal()
	}

	trie.AddRoute("GET", "/:id/:property.:format", "")

	if trie.root.Children["/"].ParamChild.Children["/"].ParamChild.Children["."].ParamChild == nil {
		t.Fatal()
	}
	if trie.root.Children["/"].ParamName != "id" {
		t.Fatal()
	}
	if trie.root.Children["/"].ParamChild.Children["/"].ParamName != "property" {
		t.Fatal()
	}
	if trie.root.Children["/"].ParamChild.Children["/"].ParamChild.Children["."].ParamName != "format" {
		t.Fatal()
	}
}

func TestRelaxedInsert(t *testing.T) {
	trie := New()

	trie.AddRoute("GET", "/#id/", "")
	if trie.root.Children["/"].RelaxedChild.Children["/"] == nil {
		t.Fatal()
	}
	if trie.root.Children["/"].RelaxedName != "id" {
		t.Fatal()
	}
}

func TestSplatInsert(t *testing.T) {
	trie := New()
	trie.AddRoute("GET", "/*splat", "")
	if trie.root.Children["/"].SplatChild == nil {
		t.Fatal()
	}
}

func TestDupeInsert(t *testing.T) {
	trie := New()
	trie.AddRoute("GET", "/", "1")
	err := trie.AddRoute("GET", "/", "2")
	if err == nil {
		t.Fatal()
	}
	if trie.root.Children["/"].HttpMethodToRoute["GET"] != "1" {
		t.Fatal()
	}
}

func isInMatches(test string, matches []*Match) bool {
	for _, match := range matches {
		if match.Route.(string) == test {
			return true
		}
	}
	return false
}

func TestFindRoute(t *testing.T) {

	trie := New()

	trie.AddRoute("GET", "/", "root")
	trie.AddRoute("GET", "/r/:id", "resource")
	trie.AddRoute("GET", "/r/:id/property", "property")
	trie.AddRoute("GET", "/r/:id/property.*format", "property_format")
	trie.AddRoute("GET", "/user/#username/property", "user_property")

	trie.Compress()

	matches := trie.FindRoutes("GET", "/")
	if len(matches) != 1 {
		t.Fatalf("expected one route, got %d", len(matches))
	}
	if !isInMatches("root", matches) {
		t.Fatal("expected 'root'")
	}

	matches = trie.FindRoutes("GET", "/notfound")
	if len(matches) != 0 {
		t.Fatalf("expected zero route, got %d", len(matches))
	}

	matches = trie.FindRoutes("GET", "/r/1")
	if len(matches) != 1 {
		t.Fatalf("expected one route, got %d", len(matches))
	}
	if !isInMatches("resource", matches) {
		t.Fatalf("expected 'resource', got %+v", matches)
	}
	if matches[0].Params["id"] != "1" {
		t.Fatal()
	}

	matches = trie.FindRoutes("GET", "/r/1/property")
	if len(matches) != 1 {
		t.Errorf("expected one route, got %d", len(matches))
	}
	if !isInMatches("property", matches) {
		t.Fatal("expected 'property'")
	}
	if matches[0].Params["id"] != "1" {
		t.Fatal()
	}

	matches = trie.FindRoutes("GET", "/r/1/property.json")
	if len(matches) != 1 {
		t.Errorf("expected one route, got %d", len(matches))
	}
	if !isInMatches("property_format", matches) {
		t.Fatal("expected 'property_format'")
	}
	if matches[0].Params["id"] != "1" {
		t.Fatal()
	}
	if matches[0].Params["format"] != "json" {
		t.Fatal()
	}

	matches = trie.FindRoutes("GET", "/user/antoine.imbert/property")
	if len(matches) != 1 {
		t.Errorf("expected one route, got %d", len(matches))
	}
	if !isInMatches("user_property", matches) {
		t.Fatal("expected 'user_property'")
	}
	if matches[0].Params["username"] != "antoine.imbert" {
		t.Fatal()
	}
}

func TestFindRouteMultipleMatches(t *testing.T) {

	trie := New()

	trie.AddRoute("GET", "/r/1", "resource1")
	trie.AddRoute("GET", "/r/2", "resource2")
	trie.AddRoute("GET", "/r/:id", "resource_generic")
	trie.AddRoute("GET", "/s/*rest", "special_all")
	trie.AddRoute("GET", "/s/:param", "special_generic")
	trie.AddRoute("GET", "/s/#param", "special_relaxed")
	trie.AddRoute("GET", "/", "root")

	trie.Compress()

	matches := trie.FindRoutes("GET", "/r/1")
	if len(matches) != 2 {
		t.Errorf("expected two matches, got %d", len(matches))
	}
	if !isInMatches("resource_generic", matches) {
		t.Fatal()
	}
	if !isInMatches("resource1", matches) {
		t.Fatal()
	}

	matches = trie.FindRoutes("GET", "/s/1")
	if len(matches) != 3 {
		t.Errorf("expected two matches, got %d", len(matches))
	}
	if !isInMatches("special_all", matches) {
		t.Fatal()
	}
	if !isInMatches("special_generic", matches) {
		t.Fatal()
	}
	if !isInMatches("special_relaxed", matches) {
		t.Fatal()
	}
}

func TestConsistentPlaceholderName(t *testing.T) {

	trie := New()

	trie.AddRoute("GET", "/r/:id", "oneph")
	err := trie.AddRoute("GET", "/r/:rid/other", "twoph")
	if err == nil {
		t.Fatal("Should have died on inconsistent placeholder name")
	}

	trie.AddRoute("GET", "/r/#id", "oneph")
	err = trie.AddRoute("GET", "/r/#rid/other", "twoph")
	if err == nil {
		t.Fatal("Should have died on inconsistent placeholder name")
	}

	trie.AddRoute("GET", "/r/*id", "oneph")
	err = trie.AddRoute("GET", "/r/*rid", "twoph")
	if err == nil {
		t.Fatal("Should have died on duplicated route")
	}
}

func TestDuplicateName(t *testing.T) {

	trie := New()

	err := trie.AddRoute("GET", "/r/:id/o/:id", "two")
	if err == nil {
		t.Fatal("Should have died, this route has two placeholder named `id`")
	}

	err = trie.AddRoute("GET", "/r/:id/o/*id", "two")
	if err == nil {
		t.Fatal("Should have died, this route has two placeholder named `id`")
	}

	err = trie.AddRoute("GET", "/r/:id/o/#id", "two")
	if err == nil {
		t.Fatal("Should have died, this route has two placeholder named `id`")
	}
}
