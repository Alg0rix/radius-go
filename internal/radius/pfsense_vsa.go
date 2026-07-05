package radius

import (
	"encoding/binary"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// pfSense vendor ID (also used by OPNsense, which is a pfSense fork).
const pfSenseVendorID uint32 = 13644

// pfSense captive portal VSA attribute numbers (all uint32 per FreeRADIUS dictionary).
const (
	pfSenseBandwidthMaxUp   byte = 1
	pfSenseBandwidthMaxDown byte = 2
	pfSenseMaxTotalOctets   byte = 3
	pfSenseDNS1             byte = 4
	pfSenseDNS2             byte = 5
)

// --- setters for RADIUS response packets ---

// PfSenseBandwidthMaxUp_Set adds the pfSense-Bandwidth-Max-Up VSA.
func PfSenseBandwidthMaxUp_Set(p *radius.Packet, value uint32) error {
	return vsaSetUint32(p, pfSenseVendorID, pfSenseBandwidthMaxUp, value)
}

// PfSenseBandwidthMaxDown_Set adds the pfSense-Bandwidth-Max-Down VSA.
func PfSenseBandwidthMaxDown_Set(p *radius.Packet, value uint32) error {
	return vsaSetUint32(p, pfSenseVendorID, pfSenseBandwidthMaxDown, value)
}

// PfSenseMaxTotalOctets_Set adds the pfSense-Max-Total-Octets VSA.
func PfSenseMaxTotalOctets_Set(p *radius.Packet, value uint32) error {
	return vsaSetUint32(p, pfSenseVendorID, pfSenseMaxTotalOctets, value)
}

// PfSenseDNS1_Set adds the pfSense-Primary-DNS VSA.
func PfSenseDNS1_Set(p *radius.Packet, ip uint32) error {
	return vsaSetUint32(p, pfSenseVendorID, pfSenseDNS1, ip)
}

// PfSenseDNS2_Set adds the pfSense-Secondary-DNS VSA.
func PfSenseDNS2_Set(p *radius.Packet, ip uint32) error {
	return vsaSetUint32(p, pfSenseVendorID, pfSenseDNS2, ip)
}

// pfSenseVSAGetUint32 extracts a uint32 VSA value from a RADIUS packet.
func pfSenseVSAGetUint32(p *radius.Packet, vendorType byte) (uint32, bool) {
	for _, avp := range p.Attributes {
		if avp.Type != rfc2865.VendorSpecific_Type {
			continue
		}
		vendorID, vsa, err := radius.VendorSpecific(avp.Attribute)
		if err != nil || vendorID != pfSenseVendorID {
			continue
		}
		for len(vsa) >= 3 {
			vt := vsa[0]
			vlen := int(vsa[1])
			if vlen < 3 || vlen > len(vsa) {
				break
			}
			if vt == vendorType && vlen >= 6 {
				return binary.BigEndian.Uint32(vsa[2:6]), true
			}
			vsa = vsa[vlen:]
		}
	}
	return 0, false
}