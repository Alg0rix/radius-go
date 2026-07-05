package radius

import (
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// MikroTik vendor ID is 14988.
const mikrotikVendorID uint32 = 14988

// MikroTik-Rate-Limit (vendor type 8) and MikroTik-Group (vendor type 4).
const (
	mikrotikRateLimitType byte = 8
	mikrotikGroupType     byte = 4
)

// MikrotikRateLimit_SetString adds a MikroTik-Rate-Limit VSA.
func MikrotikRateLimit_SetString(p *radius.Packet, value string) error {
	return mikrotikVSASetString(p, mikrotikRateLimitType, value, true)
}

// MikrotikGroup_SetString adds a MikroTik-Group VSA.
func MikrotikGroup_SetString(p *radius.Packet, value string) error {
	return mikrotikVSASetString(p, mikrotikGroupType, value, true)
}

func mikrotikVSASetString(p *radius.Packet, vendorType byte, value string, set bool) error {
	if value == "" {
		return nil
	}
	vendorAttr, err := makeMikrotikVSA(vendorType, []byte(value))
	if err != nil {
		return err
	}
	if set {
		p.Set(rfc2865.VendorSpecific_Type, vendorAttr)
	} else {
		p.Add(rfc2865.VendorSpecific_Type, vendorAttr)
	}
	return nil
}

func makeMikrotikVSA(vendorType byte, value []byte) (radius.Attribute, error) {
	sub := make([]byte, 2+len(value))
	sub[0] = vendorType
	sub[1] = byte(len(sub))
	copy(sub[2:], value)
	return radius.NewVendorSpecific(mikrotikVendorID, sub)
}

func mikrotikVSAGetString(p *radius.Packet, vendorType byte) string {
	for _, avp := range p.Attributes {
		if avp.Type != rfc2865.VendorSpecific_Type {
			continue
		}
		vendorID, vsa, err := radius.VendorSpecific(avp.Attribute)
		if err != nil || vendorID != mikrotikVendorID {
			continue
		}
		for len(vsa) >= 3 {
			vt := vsa[0]
			vlen := int(vsa[1])
			if vlen < 3 || vlen > len(vsa) {
				break
			}
			if vt == vendorType {
				return string(vsa[2:vlen])
			}
			vsa = vsa[vlen:]
		}
	}
	return ""
}