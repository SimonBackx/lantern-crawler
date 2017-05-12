package crawler

import (
	"net/url"
	"strings"
)

// ResolveReference resolves a URI reference to an absolute URI from
// an absolute base URI, per RFC 3986 Section 5.2.  The URI reference
// may be relative or absolute. ResolveReference always returns a new
// URL instance, even if the returned URL is identical to either the
// base or reference. If ref is an absolute URL, then ResolveReference
// ignores base and returns a copy of ref.
func ResolveReferenceNoCopy(base *url.URL, u *url.URL) {
	is_path := (u.Path == "")
	is_frag := (u.Fragment == "")
	is_raw := (u.RawQuery == "")
	is_secon := (u.Scheme != "" || u.Host != "" || u.User != nil)

	if u.Scheme == "" {
		u.Scheme = base.Scheme
	}

	if is_secon {
		// The "absoluteURI" or "net_path" cases.
		// We can ignore the error from setPath since we know we provided a
		// validly-escaped path.
		//u.Path = u.EscapedPath()
		return
	}

	if is_path {
		if is_raw {
			u.RawQuery = base.RawQuery
			if is_frag {
				u.Fragment = base.Fragment
			}
		}
	}
	// The "abs_path" or "rel_path" cases.
	u.Host = base.Host
	u.User = base.User
	u.Path = resolvePathNoDots(base.EscapedPath(), u.EscapedPath())
}

func resolvePathNoDots(base, ref string) string {
	if ref == "" {
		return base
	} else if ref[0] != '/' {
		i := strings.LastIndex(base, "/")
		return base[:i+1] + ref
	}
	return ref
}
