package core

import "errors"

// ErrNotFound is returned by store implementations when a requested document
// or asset does not exist. API handlers check this sentinel to return HTTP 404.
var ErrNotFound = errors.New("not found")

// ErrInvalidPath is returned by store implementations when a document or asset
// path is empty, absolute, or attempts directory traversal. API handlers check
// this sentinel to return HTTP 400.
var ErrInvalidPath = errors.New("invalid path: directory traversal not allowed")
