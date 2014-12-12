package route

import "testing"

func test() (string, error) {
	return "", nil
}

func BenchmarkFindRoutes(b *testing.B) {
	trie := New()
	trie.AddRoute("GET", "/", "root")
	trie.AddRoute("GET", "/v2/grids/:id", "1")
	trie.AddRoute("GET", "/v2/grids", "2")
	trie.AddRoute("GET", "/v2/grids", "3")
	trie.AddRoute("GET", "/r/:id", "resource")
	trie.AddRoute("GET", "/r/:id/property", "property")
	trie.AddRoute("GET", "/r/:id/property.*format", "property_format")
	trie.AddRoute("GET", "/user/#username/property", "user_property")
	trie.Compress()

	for i := 0; i < b.N; i++ {
		trie.FindRoutes("GET", "/user/verylongusernamehere/property")
	}
}
