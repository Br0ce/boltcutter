package ui

import "errors"

// errNotFound is what the fake tree answers for a path it does not
// hold.
var errNotFound = errors.New("not found")
