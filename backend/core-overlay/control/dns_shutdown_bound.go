package control

import (
	"context"
	"errors"
	"net"
)

// A graceful drain deadline is not a bind-release failure. Always close the
// actual socket; only then may reload proceed while old queries finish.
func stopDNSBind(ctx context.Context, shutdown func(context.Context) error, closeSocket func() error) error {
	drainErr := shutdown(ctx)
	closeErr := closeSocket()
	if errors.Is(closeErr, net.ErrClosed) { closeErr = nil }
	if closeErr == nil && errors.Is(drainErr, context.DeadlineExceeded) { return nil }
	return errors.Join(drainErr, closeErr)
}
