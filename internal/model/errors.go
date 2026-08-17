package model

import "errors"

// ErrDuplicateKey is returned by repository implementations when a write
// violates a unique constraint (e.g. two concurrent /staff/create calls for
// the same username in the same hospital racing past the pre-check). Lives
// here, not in repository, so service can recognize it without importing the
// concrete repository package.
var ErrDuplicateKey = errors.New("duplicate key")
