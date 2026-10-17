package repository

import "errors"

var ErrIdempotencyConflict = errors.New("idempotency key already used with different payload")
