package core

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// imgSrcRe matches <img> tags and captures the src attribute value.
// Group 1: everything up to and including src="
// Group 2: the src value
// Group 3: the closing quote
var imgSrcRe = regexp.MustCompile(`(<img\s[^>]*?src=")([^"]+)(")`)

// RewriteImageURLs rewrites relative image URLs in rendered HTML so they
// point to the asset serving route (/assets/{repo}/{resolvedPath}).
// Absolute URLs (http, https, //, data:) and already-absolute paths
// (starting with /) are left unchanged.
//
// The query string and fragment of the original src are preserved: only the
// path component is rewritten so that references like "sprite.svg#icon" or
// "img.png?raw=1" keep their semantics after rewriting.
func RewriteImageURLs(html []byte, repo, docPath string) []byte {
	docDir := path.Dir(docPath)

	return imgSrcRe.ReplaceAllFunc(html, func(match []byte) []byte {
		submatch := imgSrcRe.FindSubmatch(match)
		if len(submatch) < 4 {
			return match
		}

		src := string(submatch[2])

		if isAbsoluteURL(src) {
			return match
		}

		// Parse the src so we can rewrite only its path while preserving any
		// original query string or fragment. url.Parse also validates
		// percent-escape sequences, so malformed escapes are caught here.
		u, err := url.Parse(src)
		if err != nil {
			return match
		}

		// Resolve the path component against the document's directory.
		resolvedPath := path.Clean(path.Join(docDir, u.Path))

		// Prevent directory traversal outside the repo root.
		// Use == ".." or HasPrefix("../") to avoid false-positives on paths like
		// "..images/logo.png" that start with ".." but don't escape the root.
		if resolvedPath == ".." || strings.HasPrefix(resolvedPath, "../") {
			return match
		}

		// Build the rewritten path, percent-encoding each path segment so that
		// names containing spaces produce valid, unambiguous URLs.
		// url.JoinPath encodes each segment individually and preserves slashes.
		newPath, err := url.JoinPath("/assets/", repo, resolvedPath)
		if err != nil {
			// Malformed segments — leave the original src unchanged.
			return match
		}

		// Re-attach the original query string and fragment (if any) so that
		// references like "sprite.svg#icon" or "img.png?raw=1" keep their
		// semantics and are not double-encoded.
		newSrc := newPath
		if u.RawQuery != "" {
			newSrc += "?" + u.RawQuery
		}

		if u.Fragment != "" {
			newSrc += "#" + u.Fragment
		}

		buf := make([]byte, 0, len(submatch[1])+len(newSrc)+len(submatch[3]))
		buf = append(buf, submatch[1]...)
		buf = append(buf, []byte(newSrc)...)
		buf = append(buf, submatch[3]...)

		return buf
	})
}

// isAbsoluteURL reports whether src is an absolute URL or data URI
// that should not be rewritten.
func isAbsoluteURL(src string) bool {
	return strings.HasPrefix(src, "http://") ||
		strings.HasPrefix(src, "https://") ||
		strings.HasPrefix(src, "//") ||
		strings.HasPrefix(src, "data:") ||
		strings.HasPrefix(src, "/")
}
