package control

import (
	"errors"
	"syscall"
	"testing"
)

func TestLPMBatchENOMEMReplaysPartialBatch(t *testing.T) {
	entries := map[int]int{0: 4, 1: 5}
	n, err := retryLPMBatch(4, func() (int, error) { return 2, syscall.ENOMEM }, func(i int) error {
		entries[i] = i + 4
		return nil
	})
	if err != nil || n != 4 || len(entries) != 4 || entries[3] != 7 {
		t.Fatalf("partial batch not completed: n=%d err=%v entries=%v", n, err, entries)
	}
}

func TestLPMBatchPreservesOtherErrorsAndNativeSuccess(t *testing.T) {
	for _, result := range []error{nil, syscall.EINVAL, syscall.EPERM} {
		n, err := retryLPMBatch(4, func() (int, error) { return 2, result }, func(int) error {
			t.Fatal("successful batch or unrelated error must not be replayed")
			return nil
		})
		if n != 2 || !errors.Is(err, result) {
			t.Fatalf("native result changed: %d %v", n, err)
		}
	}
}

func TestLPMBatchReplayFailureRetainsPartialCount(t *testing.T) {
	n, err := retryLPMBatch(4, func() (int, error) { return 0, syscall.ENOMEM }, func(i int) error {
		if i == 2 { return syscall.ENOSPC }
		return nil
	})
	if n != 2 || !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("replay error hidden: %d %v", n, err)
	}
}
