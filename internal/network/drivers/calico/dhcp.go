package calico

import (
	"context"
	"net"

	"github.com/alphadose/haxmap"
	"github.com/cockroachdb/errors"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/projecteru2/core/log"
)

type singleServer struct {
	*server4.Server
	Iface string
	IP    net.IP
}

type DHCPServer struct {
	Servers haxmap.Map[string, *singleServer]
	Port    int
	gwIP    net.IP
}

func NewDHCPServer(gw net.IP) *DHCPServer {
	return &DHCPServer{
		Port: 67,
		gwIP: gw,
	}
}

func (srv *DHCPServer) AddInterface(iface string, ip net.IP, ipNet *net.IPNet) error {
	logger := log.WithFunc("DHCPServer.AddInterface")
	laddr := net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: srv.Port,
	}
	dhcpSrv, err := server4.NewServer(iface, &laddr, srv.NewHandler(ip, ipNet))
	if err != nil {
		return errors.Wrapf(err, "fialed to create dhcpv4 server")
	}
	if oldSrv, exists := srv.Servers.Get(iface); exists {
		logger.Infof(context.TODO(), "Stop old dhcp server for interface %s", iface)
		oldSrv.Close()
	}
	srv.Servers.Set(iface, &singleServer{
		Server: dhcpSrv,
		Iface:  iface,
		IP:     ip,
	})
	go func() {
		defer logger.Infof(context.TODO(), "dhcp server for %s: %s exits", iface, ip.String())
		logger.Infof(context.TODO(), "starting dhcp server for %s: %s", iface, ip.String())
		_ = dhcpSrv.Serve()
	}()
	return nil
}

func (srv *DHCPServer) RemoveInterface(iface string) {
	ssrv, exists := srv.Servers.GetAndDel(iface)
	if !exists {
		return
	}
	ssrv.Close()
}

func (srv *DHCPServer) NewHandler(ip net.IP, ipNet *net.IPNet) server4.Handler {
	return func(conn net.PacketConn, peer net.Addr, m *dhcpv4.DHCPv4) {
		// Process DHCP requests (DISCOVER, OFFER, REQUEST, DECLINE, RELEASE)
		logger := log.WithFunc("dhcp.handler")
		logger.Debugf(context.TODO(), m.Summary())
		leaseTime := uint32(3600)

		switch m.MessageType() {
		case dhcpv4.MessageTypeDiscover:
			// Offer an IP address from the subnet based on server IP
			offer, err := dhcpv4.NewReplyFromRequest(
				m,
				dhcpv4.WithMessageType(dhcpv4.MessageTypeOffer),
				dhcpv4.WithYourIP(ip),
				dhcpv4.WithClientIP(m.ClientIPAddr),
				dhcpv4.WithLeaseTime(leaseTime),
				dhcpv4.WithNetmask(ipNet.Mask),
				dhcpv4.WithGatewayIP(srv.gwIP),
			)
			if err != nil {
				logger.Errorf(context.TODO(), err, "Failed to create DHCP offer")
				return
			}
			if _, err := conn.WriteTo(offer.ToBytes(), peer); err != nil {
				logger.Error(context.TODO(), err, "failed to write offer packet.")
			}
		case dhcpv4.MessageTypeRequest:
			// Check if requested IP is within our range and send ACK
			if m.YourIPAddr.Equal(ip) {
				ack, err := dhcpv4.NewReplyFromRequest(
					m,
					dhcpv4.WithMessageType(dhcpv4.MessageTypeAck),
					dhcpv4.WithYourIP(ip),
					dhcpv4.WithClientIP(m.ClientIPAddr),
					// dhcpv4.WithServerIP(ip),
					dhcpv4.WithLeaseTime(leaseTime),
					dhcpv4.WithNetmask(ipNet.Mask),
					dhcpv4.WithGatewayIP(srv.gwIP),
				)
				if err != nil {
					logger.Errorf(context.TODO(), err, "Failed to create DHCP ACK")
				}
				if _, err := conn.WriteTo(ack.ToBytes(), peer); err != nil {
					logger.Errorf(context.TODO(), err, "failed to write ACK package.")
				}
			} else {
				logger.Warnf(context.TODO(), "Invalid IP request from %s for %s", peer, m.YourIPAddr)
			}
		default:
			logger.Warnf(context.TODO(), "Unhandled DHCP message type: %d", m.MessageType())
		}
	}
}
