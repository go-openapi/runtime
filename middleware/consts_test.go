// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package middleware

// Test-only constants extracted to satisfy the goconst linter.
// These are shared across the test files of the middleware package.

const (
	// struct field / map key names for spec.Parameter maps.
	keyID        = "ID"
	keyName      = "Name"
	keyTags      = "Tags"
	keyFriend    = "Friend"
	keyRequestID = "RequestID"
)

const (
	// lowercase spec parameter / route param keys.
	paramKeyID   = "id"
	paramKeyName = "name"
	paramKeyAge  = "age"
)

const (
	// collection format identifiers.
	multiFmt = "multi"
	pipesFmt = "pipes"
	ssvFmt   = "ssv"
	tsvFmt   = "tsv"
)

// jsonMime is the application/json content type used throughout the test suite.
// Local const (not the public runtime.JSONMime) so it stays self-contained.
const jsonMime = "application/json"

const (
	// recurring test values.
	tagOne        = "one"
	tagTwo        = "two"
	tagThree      = "three"
	valToby       = "toby"
	valHello      = "hello"
	valYada       = "yada"
	valFragment   = "fragment"
	pathSomething = "/something"
	valTheUser    = "the user"
	testAuth2     = "auth2"
	testAuth3     = "auth3"
	testAuth4     = "auth4"
)
