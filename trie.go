// Special Trie implementation for HTTP routing.
//
// This Trie implementation is designed to support strings that includes
// :param and *splat parameters. Strings that are commonly used to represent
// the Path in HTTP routing. This implementation also maintain for each Path
// a map of HTTP Methods associated with the Route.
//
// You probably don't need to use this package directly.
//
// original from https://raw.githubusercontent.com/ant0ine/go-json-rest/master/rest/trie/impl.go (MIT)
// https://github.com/squiidz/bone/blob/master/route.go (MIT)
//
package route

import "fmt"

func splitParamIndex(remaining string) int {
	i := 0
	for len(remaining) > i && remaining[i] != '/' && remaining[i] != '.' {
		i++
	}
	return i
}

func splitParam(remaining string) (string, string) {
	i := splitParamIndex(remaining)
	return remaining[:i], remaining[i:]
}

func splitRelaxedIndex(remaining string) int {
	i := 0
	for len(remaining) > i && remaining[i] != '/' {
		i++
	}
	return i
}

func splitRelaxed(remaining string) (string, string) {
	i := splitRelaxedIndex(remaining)
	return remaining[:i], remaining[i:]
}

// Index returns the index of the first instance of s in sa, or -1 if s is not present in sa.
func Index(sa []string, s string) int {
	for idx, ss := range sa {
		if ss == s {
			return idx
		}
	}
	return -1
}

func Contains(sa []string, s string) bool {
	return Index(sa, s) >= 0
}

type node struct {
	Children map[string]*node

	ChildrenKeyLen int

	ParamChild *node
	ParamName  string

	RelaxedChild *node
	RelaxedName  string

	SplatChild *node
	SplatName  string

	HttpMethodToRoute map[string]interface{}

	paramNames []string
	paramCount int
}

func (n *node) addRoute(pathExp string, r *Route) error {
	nextNode := n

	var (
		name      string
		token     string
		remaining string
	)

	for len(pathExp) != 0 {

		token = pathExp[0:1]
		remaining = pathExp[1:]

		switch token[0] {
		case ':':
			// :param case
			name, remaining = splitParam(remaining)

			// Check param name is unique
			if Contains(r.params, name) {
				return fmt.Errorf("A route can't have two placeholders with the same name: %s", name)
			}

			r.params = append(r.params, name)

			if nextNode.ParamChild == nil {
				nextNode.ParamChild = &node{}
				nextNode.ParamName = name
			} else if nextNode.ParamName != name {
				return fmt.Errorf("Routes sharing a common placeholder MUST name it consistently: %s != %s", nextNode.ParamName, name)
			}

			nextNode = nextNode.ParamChild

		case '#':
			// #param case
			name, remaining = splitRelaxed(remaining)

			// Check param name is unique
			if Contains(r.params, name) {
				return fmt.Errorf("A route can't have two placeholders with the same name: %s", name)
			}

			r.params = append(r.params, name)

			if nextNode.RelaxedChild == nil {
				nextNode.RelaxedChild = &node{}
				nextNode.RelaxedName = name
			} else if nextNode.RelaxedName != name {
				return fmt.Errorf("Routes sharing a common placeholder MUST name it consistently: %s != %s", nextNode.RelaxedName, name)
			}
			nextNode = nextNode.RelaxedChild

		case '*':
			// *splat case
			name := remaining
			remaining = ""

			// Check param name is unique
			if Contains(r.params, name) {
				return fmt.Errorf("A route can't have two placeholders with the same name: %s", name)
			}

			if nextNode.SplatChild == nil {
				nextNode.SplatChild = &node{}
				nextNode.SplatName = name
			}
			nextNode = n.SplatChild

		default:
			// general case
			if nextNode.Children == nil {
				nextNode.Children = map[string]*node{}
				nextNode.ChildrenKeyLen = 1
			}

			if nextNode.Children[token] == nil {
				nextNode.Children[token] = &node{}
			}

			nextNode = nextNode.Children[token]
		}

		pathExp = remaining
	}

	// keep track of how many params are used to be able to allocate array
	//	nextNode.paramNames = usedParams

	// // end of the path, leaf node, update the map
	// if nextNode.HttpMethodToRoute == nil {
	// 	nextNode.HttpMethodToRoute = map[string]interface{}{
	// 		httpMethod: route,
	// 	}
	// 	return nil
	// }

	// if nextNode.HttpMethodToRoute[httpMethod] != nil {
	// 	return errors.New("node.Route already set, duplicated path and method")
	// }

	// nextNode.HttpMethodToRoute[httpMethod] = route

	return nil
}

func (n *node) compress() {
	// *splat branch
	if n.SplatChild != nil {
		n.SplatChild.compress()
	}
	// :param branch
	if n.ParamChild != nil {
		n.ParamChild.compress()
	}
	// #param branch
	if n.RelaxedChild != nil {
		n.RelaxedChild.compress()
	}
	// main branch
	if len(n.Children) == 0 {
		return
	}
	// compressable ?
	canCompress := true
	for _, node := range n.Children {
		if node.HttpMethodToRoute != nil || node.SplatChild != nil || node.ParamChild != nil || node.RelaxedChild != nil {
			canCompress = false
		}
	}
	// compress
	if canCompress {
		merged := map[string]*node{}
		for key, node := range n.Children {
			for gdKey, gdNode := range node.Children {
				mergedKey := key + gdKey
				merged[mergedKey] = gdNode
			}
		}
		n.Children = merged
		n.ChildrenKeyLen++
		n.compress()
		// continue
	} else {
		for _, node := range n.Children {
			node.compress()
		}
	}
}

func printFPadding(padding int, format string, a ...interface{}) {
	for i := 0; i < padding; i++ {
		fmt.Print(" ")
	}
	fmt.Printf(format, a...)
}

// Private function for now
func (n *node) printDebug(level int) {
	level++
	// *splat branch
	if n.SplatChild != nil {
		printFPadding(level, "*splat\n")
		n.SplatChild.printDebug(level)
	}
	// :param branch
	if n.ParamChild != nil {
		printFPadding(level, ":param("+n.ParamName+")\n")
		n.ParamChild.printDebug(level)
	}
	// #param branch
	if n.RelaxedChild != nil {
		printFPadding(level, "#relaxed\n")
		n.RelaxedChild.printDebug(level)
	}
	// main branch
	for key, node := range n.Children {
		printFPadding(level, "\"%s\"\n", key)
		node.printDebug(level)
	}
}

// utility for the node.findRoutes recursive method

type paramMatch struct {
	name  string
	value string
}

type findContext struct {
	path       string
	method     string
	paramStack []paramMatch
	matchFunc  func(httpMethod, path string, node *node)
}

func newFindContext(method, path string) *findContext {
	return &findContext{
		method:     method,
		path:       path,
		paramStack: []paramMatch{},
	}
}

func (fc *findContext) pushParams(name string, value int) {
	fc.paramStack = append(
		fc.paramStack,
		//		paramMatch{name, value},
	)
}

func (fc *findContext) popParams() {
	//fc.paramStack = fc.paramStack[:len(fc.paramStack)-1]
}

func (fc *findContext) paramsAsMap() map[string]string {
	r := map[string]string{}
	// for _, param := range fc.paramStack {
	// 	if r[param.name] != "" {
	// 		// this is checked at addRoute time, and should never happen.
	// 		panic(fmt.Sprintf(
	// 			"placeholder %s already found, placeholder names should be unique per route",
	// 			param.name,
	// 		))
	// 	}
	// 	r[param.name] = param.value
	// }
	return r
}

type Match struct {
	// Same Route as in AddRoute
	Route interface{}
	// map of params matched for this result
	Params map[string]string
}

func (c *findContext) traverseTrie(t *Trie, path string) {
	c.traverseNode(t.root, path)
}

func (c *findContext) traverseNode(n *node, path string) {

	if n.HttpMethodToRoute != nil && path == "" {
		c.matchFunc(c.method, path, n)
	}

	if len(path) == 0 {
		return
	}

	// *splat branch
	if n.SplatChild != nil {
		c.pushParams(n.SplatName, len(path))
		c.traverseNode(n.SplatChild, "")
		c.popParams()
	}

	// :param branch
	if n.ParamChild != nil {
		idx := splitParamIndex(path)
		c.pushParams(n.ParamName, idx)
		c.traverseNode(n.ParamChild, path[idx:])
		c.popParams()
	}

	// #param branch
	if n.RelaxedChild != nil {
		idx := splitRelaxedIndex(path)
		c.pushParams(n.RelaxedName, idx)
		c.traverseNode(n.RelaxedChild, path[idx:])
		c.popParams()
	}

	// main branch
	length := n.ChildrenKeyLen
	if len(path) < length {
		return
	}

	token := path[0:length]
	remaining := path[length:]
	if n.Children[token] != nil {
		c.traverseNode(n.Children[token], remaining)
	}
}

type Trie struct {
	root *node
}

// Instanciate a Trie with an empty node as the root.
func New() *Trie {
	return &Trie{
		root: &node{},
	}
}

// Insert the route in the Trie following or creating the nodes corresponding to the path.
func (t *Trie) AddRoute(httpMethod, pathExp string, route interface{}) error {
	return t.root.addRoute(pathExp, &Route{Method: httpMethod})
}

// Reduce the size of the tree, must be done after the last AddRoute.
func (t *Trie) Compress() {
	t.root.compress()
}

// Private function for now.
func (t *Trie) printDebug() {
	fmt.Print("<trie>\n")
	t.root.printDebug(0)
	fmt.Print("</trie>\n")
}

// Given a path and an http method, return all the matching routes.
func (t *Trie) FindRoutes(httpMethod, path string) []*Match {
	context := newFindContext(httpMethod, path)
	matches := []*Match{}
	context.matchFunc = func(httpMethod, path string, node *node) {
		if node.HttpMethodToRoute[httpMethod] != nil {
			// path and method match, found a route !
			matches = append(
				matches,
				&Match{
					Route:  node.HttpMethodToRoute[httpMethod],
					Params: context.paramsAsMap(),
				},
			)
		}
	}
	context.traverseTrie(t, path) //.root, httpMethod, path, context)
	return matches
}

// Same as FindRoutes, but return in addition a boolean indicating if the path was matched.
// Useful to return 405
func (t *Trie) FindRoutesAndPathMatched(httpMethod, path string) ([]*Match, bool) {
	context := newFindContext(httpMethod, path)
	pathMatched := false
	matches := []*Match{}
	context.matchFunc = func(httpMethod, path string, node *node) {
		pathMatched = true
		if node.HttpMethodToRoute[httpMethod] != nil {
			// path and method match, found a route !
			matches = append(
				matches,
				&Match{
					Route:  node.HttpMethodToRoute[httpMethod],
					Params: context.paramsAsMap(),
				},
			)
		}
	}
	//find(t.root, httpMethod, path, context)
	context.traverseTrie(t, path)
	return matches, pathMatched
}

// Given a path, and whatever the http method, return all the matching routes.
func (t *Trie) FindRoutesForPath(path string) []*Match {
	context := newFindContext("", path)
	matches := []*Match{}
	context.matchFunc = func(httpMethod, path string, node *node) {
		params := context.paramsAsMap()
		for _, route := range node.HttpMethodToRoute {
			matches = append(
				matches,
				&Match{
					Route:  route,
					Params: params,
				},
			)
		}
	}
	// find(t.root, "", path, context)
	context.traverseTrie(t, path)
	return matches
}
