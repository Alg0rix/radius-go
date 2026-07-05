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
)

// --- setters for RADIUS response packets ---

// PfSenseBandwidthMaxUp_Set adds the pfSense-Bandwidth-Max-Up VSA.
func PfSenseBandwidthMaxUp_Set(p *radius.Packet, value uint32) error {
	if value == 0 {
		return nil
	}
	return pfSenseVSASetUint32(p, pfSenseBandwidthMaxUp, value)
}

// PfSenseBandwidthMaxDown_Set adds the pfSense-Bandwidth-Max-Down VSA.
func PfSenseBandwidthMaxDown_Set(p *radius.Packet, value uint32) error {
	if value == 0 {
		return nil
	}
	return pfSenseVSASetUint32(p, pfSenseBandwidthMaxDown, value)
}

// PfSenseMaxTotalOctets_Set adds the pfSense-Max-Total-Octets VSA.
func PfSenseMaxTotalOctets_Set(p *radius.Packet, value uint32) error {
	if value == 0 {
		return nil
	}
	return pfSenseVSASetUint32(p, pfSenseMaxTotalOctets, value)
}

// --- internal helpers ---

func pfSenseVSASetUint32(p *radius.Packet, vendorType byte, value uint32) error {
	attr, err := makePfSenseVSA(vendorType, value)
	if err != nil {
		return err
	}
	p.Add(rfc2865.VendorSpecific_Type, attr)
	return nil
}

// makePfSenseVSA builds a vendor-specific attribute for uint32 values.
// Wire format: vendor-type (1 byte) + length (1 byte) + 4-byte big-endian uint32.
func makePfSenseVSA(vendorType byte, value uint32) (radius.Attribute, error) {
	sub := make([]byte, 6) // 1(vendor-type) + 1(length) + 4(value)
	sub[0] = vendorType
	sub[1] = 6
	binary.BigEndian.PutUint32(sub[2:], value)
	return radius.NewVendorSpecific(pfSenseVendorID, sub)
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