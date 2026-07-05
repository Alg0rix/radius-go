package radius

import (
	"encoding/binary"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// vsaSetUint32 adds a vendor-specific uint32 sub-attribute to a RADIUS packet.
// Wire format: vendor-type (1 byte) + length (1 byte) + 4-byte big-endian value.
func vsaSetUint32(p *radius.Packet, vendorID uint32, vendorType byte, value uint32) error {
	if value == 0 {
		return nil
	}
	sub := make([]byte, 6)
	sub[0] = vendorType
	sub[1] = 6
	binary.BigEndian.PutUint32(sub[2:], value)
	attr, err := radius.NewVendorSpecific(vendorID, sub)
	if err != nil {
		return err
	}
	p.Add(rfc2865.VendorSpecific_Type, attr)
	return nil
}
