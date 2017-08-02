package commons

import (
	"fmt"

	"github.com/gdey/jsonpath"
	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

type OIDData struct {
	Oid        value.OID
	Type       pdu.VariableType
	Url        string
	Jsonpath   string
	Applicator jsonpath.Applicator
}

func NewOIDData(oid string, oid_type string, url string, jpath string) (*OIDData, error) {
	real_oid, err := ParseOID(oid)
	if err != nil {
		return nil, err
	}

	real_type, err := string2VariableType(oid_type)
	if err != nil {
		return nil, err
	}

	applicator, err := jsonpath.Parse(jpath)
	if err != nil {
		return nil, err
	}

	oid_data := OIDData{
		Oid:        real_oid,
		Type:       real_type,
		Url:        url,
		Jsonpath:   jpath,
		Applicator: applicator,
	}

	return &oid_data, nil
}

func string2VariableType(s string) (pdu.VariableType, error) {
	switch s {
	case "Integer":
		return pdu.VariableTypeInteger, nil
	case "OctetString":
		return pdu.VariableTypeOctetString, nil
	case "ObjectIdentifier":
		return pdu.VariableTypeObjectIdentifier, nil
	case "IpAddress":
		return pdu.VariableTypeIPAddress, nil
	case "Counter32":
		return pdu.VariableTypeCounter32, nil
	case "Gauge32":
		return pdu.VariableTypeGauge32, nil
	case "TimeTicks":
		return pdu.VariableTypeTimeTicks, nil
	case "Opaque":
		return pdu.VariableTypeOpaque, nil
	case "Counter64":
		return pdu.VariableTypeCounter64, nil
	}

	return pdu.VariableTypeNoSuchObject, fmt.Errorf("%v is not  a valid pdu type", s)
}

func variableType2String(t pdu.VariableType) (string, error) {
	switch t {
	case pdu.VariableTypeInteger:
		return "Integer", nil
	case pdu.VariableTypeOctetString:
		return "OctetString", nil
	case pdu.VariableTypeObjectIdentifier:
		return "ObjectIdentifier", nil
	case pdu.VariableTypeIPAddress:
		return "IpAddress", nil
	case pdu.VariableTypeCounter32:
		return "Counter32", nil
	case pdu.VariableTypeGauge32:
		return "Gauge32", nil
	case pdu.VariableTypeTimeTicks:
		return "TimeTicks", nil
	case pdu.VariableTypeOpaque:
		return "Opaque", nil
	case pdu.VariableTypeCounter64:
		return "Counter64", nil
	}

	return "", fmt.Errorf("%v is not  a valid type", t)
}
