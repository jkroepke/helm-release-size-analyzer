package releasesecret

import "errors"

var (
	errNilSecret      = errors.New("release secret is nil")
	errMissingPayload = errors.New("release secret has no release payload")
	errInvalidJSON    = errors.New("release secret contains an invalid release JSON payload")
)
