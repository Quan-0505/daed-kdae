package consoleapi

import (
	"context"
	"github.com/daeuniverse/dae-wing/dae"
	"github.com/daeuniverse/dae/common/consts"
	"github.com/daeuniverse/dae/control"
	"github.com/mzz2017/softwind/netproxy"
	"net"
	"net/netip"
	"strconv"
)

// The active plane's direct dialer carries its bypass mark, so disabling proxy
// does not accidentally send the download through transparent routing again.
func geoDirectDial(ctx context.Context, network, address string) (net.Conn, error) {
	ctl, err := dae.ControlPlane()
	if err != nil {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, &net.DNSError{Err: "no address", Name: host}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := ctl.RouteDialTcp(&control.RouteDialParam{Outbound: consts.OutboundDirect, Domain: host, Src: netip.MustParseAddrPort("0.0.0.0:0"), Dest: netip.AddrPortFrom(ips[0], uint16(port))})
	if err != nil {
		return nil, err
	}
	return &netproxy.FakeNetConn{Conn: conn}, nil
}
