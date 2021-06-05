// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// at https://github.com/julienschmidt/httprouter/blob/master/LICENSE

package gin

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Used as a workaround since we can't compare functions or their addresses
var fakeHandlerValue string

func fakeHandler(val string) HandlersChain {
	return HandlersChain{func(c *Context) {
		fakeHandlerValue = val
	}}
}

type testRequests []struct {
	path       string
	nilHandler bool
	route      string
	ps         Params
}

func getParams() *Params {
	ps := make(Params, 0, 20)
	return &ps
}

func checkRequests(t *testing.T, tree *node, requests testRequests, unescapes ...bool) {
	unescape := false
	if len(unescapes) >= 1 {
		unescape = unescapes[0]
	}

	for _, request := range requests {
		value := tree.getValue(request.path, getParams(), unescape)

		if value.handlers == nil {
			if !request.nilHandler {
				t.Errorf("handle mismatch for route '%s': Expected non-nil handle", request.path)
			}
		} else if request.nilHandler {
			t.Errorf("handle mismatch for route '%s': Expected nil handle", request.path)
		} else {
			value.handlers[0](nil)
			if fakeHandlerValue != request.route {
				t.Errorf("handle mismatch for route '%s': Wrong handle (%s != %s)", request.path, fakeHandlerValue, request.route)
			}
		}

		if value.params != nil {
			if !reflect.DeepEqual(*value.params, request.ps) {
				t.Errorf("Params mismatch for route '%s'", request.path)
			}
		}

	}
}

func checkPriorities(t *testing.T, n *node) uint32 {
	var prio uint32
	for i := range n.children {
		prio += checkPriorities(t, n.children[i])
	}

	if n.handlers != nil {
		prio++
	}

	if n.priority != prio {
		t.Errorf(
			"priority mismatch for node '%s': is %d, should be %d",
			n.path, n.priority, prio,
		)
	}

	return prio
}

func TestCountParams(t *testing.T) {
	if countParams("/path/:param1/static/*catch-all") != 2 {
		t.Fail()
	}
	if countParams(strings.Repeat("/:param", 256)) != 256 {
		t.Fail()
	}
}

func TestTreeAddAndGet(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}
	for _, route := range routes {
		tree.addRoute(route, fakeHandler(route))
	}

	checkRequests(t, tree, testRequests{
		{"/a", false, "/a", nil},
		{"/", true, "", nil},
		{"/hi", false, "/hi", nil},
		{"/contact", false, "/contact", nil},
		{"/co", false, "/co", nil},
		{"/con", true, "", nil},  // key mismatch
		{"/cona", true, "", nil}, // key mismatch
		{"/no", true, "", nil},   // no matching child
		{"/ab", false, "/ab", nil},
		{"/α", false, "/α", nil},
		{"/β", false, "/β", nil},
	})

	checkPriorities(t, tree)
}

func TestTreeWildcard(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/cmd/:tool/",
		"/cmd/:tool/:sub",
		"/cmd/whoami",
		"/cmd/whoami/root",
		"/cmd/whoami/root/",
		"/src/*filepath",
		"/search/",
		"/search/:query",
		"/search/gin-gonic",
		"/search/google",
		"/user_:name",
		"/user_:name/about",
		"/files/:dir/*filepath",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/info/:user/public",
		"/info/:user/project/:project",
		"/info/:user/project/golang",
	}
	for _, route := range routes {
		tree.addRoute(route, fakeHandler(route))
	}

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/cmd/test", true, "/cmd/:tool/", Params{Param{"tool", "test"}}},
		{"/cmd/test/", false, "/cmd/:tool/", Params{Param{"tool", "test"}}},
		{"/cmd/test/3", false, "/cmd/:tool/:sub", Params{Param{Key: "tool", Value: "test"}, Param{Key: "sub", Value: "3"}}},
		{"/cmd/who", true, "/cmd/:tool/", Params{Param{"tool", "who"}}},
		{"/cmd/who/", false, "/cmd/:tool/", Params{Param{"tool", "who"}}},
		{"/cmd/whoami", false, "/cmd/whoami", nil},
		{"/cmd/whoami/", true, "/cmd/whoami", nil},
		{"/cmd/whoami/r", false, "/cmd/:tool/:sub", Params{Param{Key: "tool", Value: "whoami"}, Param{Key: "sub", Value: "r"}}},
		{"/cmd/whoami/r/", true, "/cmd/:tool/:sub", Params{Param{Key: "tool", Value: "whoami"}, Param{Key: "sub", Value: "r"}}},
		{"/cmd/whoami/root", false, "/cmd/whoami/root", nil},
		{"/cmd/whoami/root/", false, "/cmd/whoami/root/", nil},
		{"/src/", false, "/src/*filepath", Params{Param{Key: "filepath", Value: "/"}}},
		{"/src/some/file.png", false, "/src/*filepath", Params{Param{Key: "filepath", Value: "/some/file.png"}}},
		{"/search/", false, "/search/", nil},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", Params{Param{Key: "query", Value: "someth!ng+in+ünìcodé"}}},
		{"/search/someth!ng+in+ünìcodé/", true, "", Params{Param{Key: "query", Value: "someth!ng+in+ünìcodé"}}},
		{"/search/gin", false, "/search/:query", Params{Param{"query", "gin"}}},
		{"/search/gin-gonic", false, "/search/gin-gonic", nil},
		{"/search/google", false, "/search/google", nil},
		{"/user_gopher", false, "/user_:name", Params{Param{Key: "name", Value: "gopher"}}},
		{"/user_gopher/about", false, "/user_:name/about", Params{Param{Key: "name", Value: "gopher"}}},
		{"/files/js/inc/framework.js", false, "/files/:dir/*filepath", Params{Param{Key: "dir", Value: "js"}, Param{Key: "filepath", Value: "/inc/framework.js"}}},
		{"/info/gordon/public", false, "/info/:user/public", Params{Param{Key: "user", Value: "gordon"}}},
		{"/info/gordon/project/go", false, "/info/:user/project/:project", Params{Param{Key: "user", Value: "gordon"}, Param{Key: "project", Value: "go"}}},
		{"/info/gordon/project/golang", false, "/info/:user/project/golang", Params{Param{Key: "user", Value: "gordon"}}},
	})

	checkPriorities(t, tree)
}

func TestUnescapeParameters(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/cmd/:tool/:sub",
		"/cmd/:tool/",
		"/src/*filepath",
		"/search/:query",
		"/files/:dir/*filepath",
		"/info/:user/project/:project",
		"/info/:user",
	}
	for _, route := range routes {
		tree.addRoute(route, fakeHandler(route))
	}

	unescape := true
	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/cmd/test/", false, "/cmd/:tool/", Params{Param{Key: "tool", Value: "test"}}},
		{"/cmd/test", true, "", Params{Param{Key: "tool", Value: "test"}}},
		{"/src/some/file.png", false, "/src/*filepath", Params{Param{Key: "filepath", Value: "/some/file.png"}}},
		{"/src/some/file+test.png", false, "/src/*filepath", Params{Param{Key: "filepath", Value: "/some/file test.png"}}},
		{"/src/some/file++++%%%%test.png", false, "/src/*filepath", Params{Param{Key: "filepath", Value: "/some/file++++%%%%test.png"}}},
		{"/src/some/file%2Ftest.png", false, "/src/*filepath", Params{Param{Key: "filepath", Value: "/some/file/test.png"}}},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", Params{Param{Key: "query", Value: "someth!ng in ünìcodé"}}},
		{"/info/gordon/project/go", false, "/info/:user/project/:project", Params{Param{Key: "user", Value: "gordon"}, Param{Key: "project", Value: "go"}}},
		{"/info/slash%2Fgordon", false, "/info/:user", Params{Param{Key: "user", Value: "slash/gordon"}}},
		{"/info/slash%2Fgordon/project/Project%20%231", false, "/info/:user/project/:project", Params{Param{Key: "user", Value: "slash/gordon"}, Param{Key: "project", Value: "Project #1"}}},
		{"/info/slash%%%%", false, "/info/:user", Params{Param{Key: "user", Value: "slash%%%%"}}},
		{"/info/slash%%%%2Fgordon/project/Project%%%%20%231", false, "/info/:user/project/:project", Params{Param{Key: "user", Value: "slash%%%%2Fgordon"}, Param{Key: "project", Value: "Project%%%%20%231"}}},
	}, unescape)

	checkPriorities(t, tree)
}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}

type testRoute struct {
	path     string
	conflict bool
}

func testRoutes(t *testing.T, routes []testRoute) {
	tree := &node{}

	for k := range routes {
		recv := catchPanic(func() {
			tree.addRoute(routes[k].path, nil)
		})

		if routes[k].conflict {
			if recv == nil {
				t.Errorf("no panic for conflicting route '%s'", routes[k].path)
			}
		} else if recv != nil {
			t.Errorf("unexpected panic for route '%s': %v", routes[k].path, recv)
		}
	}
}

func TestTreeWildcardConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/:tool/:sub", false},
		{"/cmd/vet", false},
		{"/foo/bar", false},
		{"/foo/:name", false},
		{"/foo/:names", true},
		{"/cmd/*path", true},
		{"/cmd/:badvar", true},
		{"/cmd/:tool/names", false},
		{"/cmd/:tool/:badsub/details", true},
		{"/src/*filepath", false},
		{"/src/:file", true},
		{"/src/static.json", true},
		{"/src/*filepathx", true},
		{"/src/", true},
		{"/src/foo/bar", true},
		{"/src1/", false},
		{"/src1/*filepath", true},
		{"/src2*filepath", true},
		{"/src2/*filepath", false},
		{"/search/:query", false},
		{"/search/valid", false},
		{"/user_:name", false},
		{"/user_x", false},
		{"/user_:name", false},
		{"/id:id", false},
		{"/id/:id", false},
	}
	testRoutes(t, routes)
}

func TestCatchAllAfterSlash(t *testing.T) {
	routes := []testRoute{
		{"/non-leading-*catchall", true},
	}
	testRoutes(t, routes)
}

func TestTreeChildConflict(t *testing.T) {
	routes := []testRoute{
		{"/cmd/vet", false},
		{"/cmd/:tool", false},
		{"/cmd/:tool/:sub", false},
		{"/cmd/:tool/misc", false},
		{"/cmd/:tool/:othersub", true},
		{"/src/AUTHORS", false},
		{"/src/*filepath", true},
		{"/user_x", false},
		{"/user_:name", false},
		{"/id/:id", false},
		{"/id:id", false},
		{"/:id", false},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeDupliatePath(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/",
		"/doc/",
		"/src/*filepath",
		"/search/:query",
		"/user_:name",
	}
	for k := range routes {
		recv := catchPanic(func() {
			tree.addRoute(routes[k], fakeHandler(routes[k]))
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", routes[k], recv)
		}

		// Add again
		recv = catchPanic(func() {
			tree.addRoute(routes[k], nil)
		})
		if recv == nil {
			t.Fatalf("no panic while inserting duplicate route '%s", routes[k])
		}
	}

	//printChildren(tree, "")

	checkRequests(t, tree, testRequests{
		{"/", false, "/", nil},
		{"/doc/", false, "/doc/", nil},
		{"/src/some/file.png", false, "/src/*filepath", Params{Param{"filepath", "/some/file.png"}}},
		{"/search/someth!ng+in+ünìcodé", false, "/search/:query", Params{Param{"query", "someth!ng+in+ünìcodé"}}},
		{"/user_gopher", false, "/user_:name", Params{Param{"name", "gopher"}}},
	})
}

func TestEmptyWildcardName(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/user:",
		"/user:/",
		"/cmd/:/",
		"/src/*",
	}
	for k := range routes {
		if catchPanic(func() { tree.addRoute(routes[k], nil) }) == nil {
			t.Fatalf("no panic while inserting route with empty wildcard name '%s", routes[k])
		}
	}
}

func TestTreeCatchAllConflict(t *testing.T) {
	routes := []testRoute{
		{"/src/*filepath/x", true},
		{"/src2/", false},
		{"/src2/*filepath/x", true},
		{"/src3/*filepath", false},
		{"/src3/*filepath/x", true},
	}
	testRoutes(t, routes)
}

func TestTreeCatchAllConflictRoot(t *testing.T) {
	routes := []testRoute{
		{"/", false},
		{"/*filepath", true},
	}
	testRoutes(t, routes)
}

func TestTreeCatchMaxParams(t *testing.T) {
	tree := &node{}
	var route = "/cmd/*filepath"
	tree.addRoute(route, fakeHandler(route))
}

func TestTreeDoubleWildcard(t *testing.T) {
	const panicMsg = "only one wildcard per path segment is allowed"

	routes := []string{
		"/:foo:bar",
		"/:foo:bar/",
		"/:foo*bar",
	}

	for key := range routes {
		tree := &node{}
		recv := catchPanic(func() {
			tree.addRoute(routes[key], nil)
		})

		if rs, ok := recv.(string); !ok || !strings.HasPrefix(rs, panicMsg) {
			t.Fatalf(`"Expected panic "%s" for route '%s', got "%v"`, panicMsg, routes[key], recv)
		}
	}
}

/*func TestTreeDuplicateWildcard(t *testing.T) {
	tree := &node{}
	routes := [...]string{
		"/:id/:name/:id",
	}
	for k := range routes {
		...
	}
}*/

func TestTreeTrailingSlashRedirect(t *testing.T) {
	tree := &node{}

	routes := [...]string{
		"/hi",
		"/b/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/admin",
		"/admin/:category",
		"/admin/:category/:page",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/no/a",
		"/no/b",
		"/api/hello/:name",
	}
	for key := range routes {
		recv := catchPanic(func() {
			tree.addRoute(routes[key], fakeHandler(routes[key]))
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", routes[key], recv)
		}
	}

	tsrRoutes := [...]string{
		"/hi/",
		"/b",
		"/search/gopher/",
		"/cmd/vet",
		"/src",
		"/x/",
		"/y",
		"/0/go/",
		"/1/go",
		"/a",
		"/admin/",
		"/admin/config/",
		"/admin/config/permissions/",
		"/doc/",
	}
	for _, route := range tsrRoutes {
		value := tree.getValue(route, nil, false)
		if value.handlers != nil {
			t.Fatalf("non-nil handler for TSR route '%s", route)
		} else if !value.tsr {
			t.Errorf("expected TSR recommendation for route '%s'", route)
		}
	}

	noTsrRoutes := [...]string{
		"/",
		"/no",
		"/no/",
		"/_",
		"/_/",
		"/api/world/abc",
	}
	for _, route := range noTsrRoutes {
		value := tree.getValue(route, nil, false)
		if value.handlers != nil {
			t.Fatalf("non-nil handler for No-TSR route '%s", route)
		} else if value.tsr {
			t.Errorf("expected no TSR recommendation for route '%s'", route)
		}
	}
}

func TestTreeRootTrailingSlashRedirect(t *testing.T) {
	tree := &node{}

	recv := catchPanic(func() {
		tree.addRoute("/:test", fakeHandler("/:test"))
	})
	if recv != nil {
		t.Fatalf("panic inserting test route: %v", recv)
	}

	value := tree.getValue("/", nil, false)
	if value.handlers != nil {
		t.Fatalf("non-nil handler")
	} else if value.tsr {
		t.Errorf("expected no TSR recommendation")
	}
}

func TestTreeFindCaseInsensitivePath(t *testing.T) {
	tree := &node{}

	longPath := "/l" + strings.Repeat("o", 128) + "ng"
	lOngPath := "/l" + strings.Repeat("O", 128) + "ng/"

	routes := [...]string{
		"/hi",
		"/b/",
		"/ABC/",
		"/search/:query",
		"/cmd/:tool/",
		"/src/*filepath",
		"/x",
		"/x/y",
		"/y/",
		"/y/z",
		"/0/:id",
		"/0/:id/1",
		"/1/:id/",
		"/1/:id/2",
		"/aa",
		"/a/",
		"/doc",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/doc/go/away",
		"/no/a",
		"/no/b",
		"/Π",
		"/u/apfêl/",
		"/u/äpfêl/",
		"/u/öpfêl",
		"/v/Äpfêl/",
		"/v/Öpfêl",
		"/w/♬",  // 3 byte
		"/w/♭/", // 3 byte, last byte differs
		"/w/𠜎",  // 4 byte
		"/w/𠜏/", // 4 byte
		longPath,
	}

	for key := range routes {
		recv := catchPanic(func() {
			tree.addRoute(routes[key], fakeHandler(routes[key]))
		})
		if recv != nil {
			t.Fatalf("panic inserting route '%s': %v", routes[key], recv)
		}
	}

	// Check out == in for all registered routes
	// With fixTrailingSlash = true
	for key := range routes {
		out, found := tree.findCaseInsensitivePath(routes[key], true)
		if !found {
			t.Errorf("Route '%s' not found!", routes[key])
		} else if string(out) != routes[key] {
			t.Errorf("Wrong result for route '%s': %s", routes[key], string(out))
		}
	}
	// With fixTrailingSlash = false
	for key := range routes {
		out, found := tree.findCaseInsensitivePath(routes[key], false)
		if !found {
			t.Errorf("Route '%s' not found!", routes[key])
		} else if string(out) != routes[key] {
			t.Errorf("Wrong result for route '%s': %s", routes[key], string(out))
		}
	}

	tests := []struct {
		in    string
		out   string
		found bool
		slash bool
	}{
		{"/HI", "/hi", true, false},
		{"/HI/", "/hi", true, true},
		{"/B", "/b/", true, true},
		{"/B/", "/b/", true, false},
		{"/abc", "/ABC/", true, true},
		{"/abc/", "/ABC/", true, false},
		{"/aBc", "/ABC/", true, true},
		{"/aBc/", "/ABC/", true, false},
		{"/abC", "/ABC/", true, true},
		{"/abC/", "/ABC/", true, false},
		{"/SEARCH/QUERY", "/search/QUERY", true, false},
		{"/SEARCH/QUERY/", "/search/QUERY", true, true},
		{"/CMD/TOOL/", "/cmd/TOOL/", true, false},
		{"/CMD/TOOL", "/cmd/TOOL/", true, true},
		{"/SRC/FILE/PATH", "/src/FILE/PATH", true, false},
		{"/x/Y", "/x/y", true, false},
		{"/x/Y/", "/x/y", true, true},
		{"/X/y", "/x/y", true, false},
		{"/X/y/", "/x/y", true, true},
		{"/X/Y", "/x/y", true, false},
		{"/X/Y/", "/x/y", true, true},
		{"/Y/", "/y/", true, false},
		{"/Y", "/y/", true, true},
		{"/Y/z", "/y/z", true, false},
		{"/Y/z/", "/y/z", true, true},
		{"/Y/Z", "/y/z", true, false},
		{"/Y/Z/", "/y/z", true, true},
		{"/y/Z", "/y/z", true, false},
		{"/y/Z/", "/y/z", true, true},
		{"/Aa", "/aa", true, false},
		{"/Aa/", "/aa", true, true},
		{"/AA", "/aa", true, false},
		{"/AA/", "/aa", true, true},
		{"/aA", "/aa", true, false},
		{"/aA/", "/aa", true, true},
		{"/A/", "/a/", true, false},
		{"/A", "/a/", true, true},
		{"/DOC", "/doc", true, false},
		{"/DOC/", "/doc", true, true},
		{"/NO", "", false, true},
		{"/DOC/GO", "", false, true},
		{"/π", "/Π", true, false},
		{"/π/", "/Π", true, true},
		{"/u/ÄPFÊL/", "/u/äpfêl/", true, false},
		{"/u/ÄPFÊL", "/u/äpfêl/", true, true},
		{"/u/ÖPFÊL/", "/u/öpfêl", true, true},
		{"/u/ÖPFÊL", "/u/öpfêl", true, false},
		{"/v/äpfêL/", "/v/Äpfêl/", true, false},
		{"/v/äpfêL", "/v/Äpfêl/", true, true},
		{"/v/öpfêL/", "/v/Öpfêl", true, true},
		{"/v/öpfêL", "/v/Öpfêl", true, false},
		{"/w/♬/", "/w/♬", true, true},
		{"/w/♭", "/w/♭/", true, true},
		{"/w/𠜎/", "/w/𠜎", true, true},
		{"/w/𠜏", "/w/𠜏/", true, true},
		{lOngPath, longPath, true, true},
	}
	// With fixTrailingSlash = true
	for _, test := range tests {
		out, found := tree.findCaseInsensitivePath(test.in, true)
		if found != test.found || (found && (string(out) != test.out)) {
			t.Errorf("Wrong result for '%s': got %s, %t; want %s, %t",
				test.in, string(out), found, test.out, test.found)
			return
		}
	}
	// With fixTrailingSlash = false
	for _, test := range tests {
		out, found := tree.findCaseInsensitivePath(test.in, false)
		if test.slash {
			if found { // test needs a trailingSlash fix. It must not be found!
				t.Errorf("Found without fixTrailingSlash: %s; got %s", test.in, string(out))
			}
		} else {
			if found != test.found || (found && (string(out) != test.out)) {
				t.Errorf("Wrong result for '%s': got %s, %t; want %s, %t",
					test.in, string(out), found, test.out, test.found)
				return
			}
		}
	}
}

func TestTreeInvalidNodeType(t *testing.T) {
	const panicMsg = "invalid node type"

	tree := &node{}
	tree.addRoute("/", fakeHandler("/"))
	tree.addRoute("/:page", fakeHandler("/:page"))

	// set invalid node type
	tree.children[0].nType = 42

	// normal lookup
	recv := catchPanic(func() {
		tree.getValue("/test", nil, false)
	})
	if rs, ok := recv.(string); !ok || rs != panicMsg {
		t.Fatalf("Expected panic '"+panicMsg+"', got '%v'", recv)
	}

	// case-insensitive lookup
	recv = catchPanic(func() {
		tree.findCaseInsensitivePath("/test", true)
	})
	if rs, ok := recv.(string); !ok || rs != panicMsg {
		t.Fatalf("Expected panic '"+panicMsg+"', got '%v'", recv)
	}
}

func TestTreeWildcardConflictEx(t *testing.T) {
	conflicts := [...]struct {
		route        string
		segPath      string
		existPath    string
		existSegPath string
	}{
		{"/who/are/foo", "/foo", `/who/are/\*you`, `/\*you`},
		{"/who/are/foo/", "/foo/", `/who/are/\*you`, `/\*you`},
		{"/who/are/foo/bar", "/foo/bar", `/who/are/\*you`, `/\*you`},
		{"/con:nection", ":nection", `/con:tact`, `:tact`},
	}

	for outKey := range conflicts {
		// I have to re-create a 'tree', because the 'tree' will be
		// in an inconsistent state when the loop recovers from the
		// panic which threw by 'addRoute' function.
		tree := &node{}
		routes := [...]string{
			"/con:tact",
			"/who/are/*you",
			"/who/foo/hello",
		}

		for k := range routes {
			tree.addRoute(routes[k], fakeHandler(routes[k]))
		}

		recv := catchPanic(func() {
			tree.addRoute(conflicts[outKey].route, fakeHandler(conflicts[outKey].route))
		})

		if !regexp.MustCompile(
			fmt.Sprintf("'%s' in new path .* conflicts with existing wildcard '%s' in existing prefix '%s'",
				conflicts[outKey].segPath, conflicts[outKey].existSegPath, conflicts[outKey].existPath)).
			MatchString(fmt.Sprint(recv)) {
			t.Fatalf("invalid wildcard conflict error (%v)", recv)
		}
	}
}
