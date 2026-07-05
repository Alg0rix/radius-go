package radius

import (
	"encoding/binary"
	"net"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

const microsoftVendorID uint32 = 311

const (
	microsoftPrimaryDNSServer   byte = 28
	microsoftSecondaryDNSServer byte = 29
)

func FramedProtocol_AddPPP(p *radius.Packet) error {
	return rfc2865.FramedProtocol_Add(p, rfc2865.FramedProtocol_Value_PPP)
}

func FramedPool_SetString(p *radius.Packet, pool string) error {
	if pool == "" {
		return nil
	}
	return rfc2869.FramedPool_SetString(p, pool)
}

func FramedIPNetmask_Add(p *radius.Packet, mask string) error {
	if mask == "" {
		return nil
	}
	ip := net.ParseIP(mask)
	if ip == nil {
		return nil
	}
	return rfc2865.FramedIPNetmask_Add(p, ip)
}

func FramedMTU_Set(p *radius.Packet, mtu int) error {
	if mtu <= 0 {
		return nil
	}
	return rfc2865.FramedMTU_Set(p, rfc2865.FramedMTU(mtu))
}

func FramedCompression_AddStac(p *radius.Packet) error {
	return rfc2865.FramedCompression_Add(p, rfc2865.FramedCompression_Value_StacLZS)
}

func MicrosoftDNS1_Set(p *radius.Packet, ip uint32) error {
	return vsaSetUint32(p, microsoftVendorID, microsoftPrimaryDNSServer, ip)
}

func MicrosoftDNS2_Set(p *radius.Packet, ip uint32) error {
	return vsaSetUint32(p, microsoftVendorID, microsoftSecondaryDNSServer, ip)
}

func emitDNS(p *radius.Packet, primary, secondary string) {
	if primary != "" {
		if ip := net.ParseIP(primary); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				v := ipToUint32(v4)
				MicrosoftDNS1_Set(p, v)
				PfSenseDNS1_Set(p, v)
			}
		}
	}
	if secondary != "" {
		if ip := net.ParseIP(secondary); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				v := ipToUint32(v4)
				MicrosoftDNS2_Set(p, v)
				PfSenseDNS2_Set(p, v)
			}
		}
	}
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}
