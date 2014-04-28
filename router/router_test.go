// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

package router

import (
	"net/http/httputil"
	"net/url"
	"testing"
)

type Registrant struct {
	label  string
	domain string
	path   string
	prefix string
	lookup string
}

var registrants = []Registrant{
	{
		label:  "foo",
		domain: "foo.bar.baz",
		path:   "foo.bar.baz/test",
		prefix: "/foo",
		lookup: "/foo/test/abc",
	},
}

func register(router *Router) {
	for _, r := range registrants {
		url, _ := url.Parse(r.path)
		proxy := httputil.NewSingleHostReverseProxy(url)

		router.Register(r.label, r.domain, r.prefix, proxy)
	}
}

func TestNew(t *testing.T) {
	router := NewRouter()
	if len(router.hosts) > 0 {
		t.Error("A new router should not contain any entries")
	}
}

func TestRegister(t *testing.T) {
	router := NewRouter()

	register(router)

	if len(router.hosts) != len(registrants) {
		t.Error("Router did not register all hosts")
	}
}

func TestLookup(t *testing.T) {
	router := NewRouter()

	register(router)

	for _, r := range registrants {
		host := router.Lookup(r.lookup)

		if host.domain != r.domain {
			t.Errorf("Lookup failed for %s, returned %s", r.lookup, host.domain)
		}
	}
}
