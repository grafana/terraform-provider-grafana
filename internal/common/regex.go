package common

import "regexp"

var (
	IDRegexp     = regexp.MustCompile(`^\d+$`)
	UIDRegexp    = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	EmailRegexp  = regexp.MustCompile(`.+\@.+\..+`)
	SHA256Regexp = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)
)
