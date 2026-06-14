package browsercookie

import "errors"

// ErrNoSessionCookie is returned when no session cookie is found for the host.
var ErrNoSessionCookie = errors.New("no session cookie found for host")
