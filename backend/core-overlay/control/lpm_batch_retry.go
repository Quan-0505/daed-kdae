package control

import (
	"errors"
	"syscall"
)

// retryLPMBatch preserves successful native batches and retries only ENOMEM.
// The caller must use UpdateAny: a failed batch may have installed a prefix,
// so replaying all entries must be idempotent. A failed replay remains an error.
func retryLPMBatch(count int, batch func() (int, error), update func(int) error) (int, error) {
	n, err := batch()
	if !errors.Is(err, syscall.ENOMEM) {
		return n, err
	}
	for i := 0; i < count; i++ {
		if err := update(i); err != nil {
			return i, err
		}
	}
	return count, nil
}
