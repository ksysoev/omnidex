package core

import (
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

		// Resolve relative path against the document's directory.
		resolved := path.Clean(path.Join(docDir, src))

		// Prevent directory traversal outside the repo root.
		if strings.HasPrefix(resolved, "..") {
			return match
		}

		newSrc := "/assets/" + repo + "/" + resolved

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
