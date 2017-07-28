package snmp_subagent

import (
	"testing"

	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

func TestBuildOidData(t *testing.T) {
	test_data := []struct {
		oid               string
		oid_type          string
		url               string
		jsonpath          string
		expected_error    bool
		expected_oid      value.OID
		expected_oid_type pdu.VariableType
	}{
		{"1.3.6.10", "OctetString", "example.com", `$["hola"]`, false, value.MustParseOID("1.3.6.10"), pdu.VariableTypeOctetString},
		{".1.3.6.10", "OctetString", "example.com", `$.first`, false, value.MustParseOID("1.3.6.10"), pdu.VariableTypeOctetString},
		{"1.3.6.10", "OctetString", "example.com", `$.second`, false, value.MustParseOID("1.3.6.10"), pdu.VariableTypeOctetString},
		{"1.3.6.10.", "OctetString", "example.com", `$["hola"]`, true, nil, pdu.VariableTypeOctetString},
		{"1.3.6.10", "OctetStringgg", "example.com", `$["hola"]`, true, nil, pdu.VariableTypeOctetString},
		{"1.3.6.10", "OctetString", "example.com", `$[["hola"]]`, true, nil, pdu.VariableTypeOctetString},
	}

	for _, test_row := range test_data {
		oid_data, err := NewOIDData(test_row.oid, test_row.oid_type, test_row.url, test_row.jsonpath)

		if test_row.expected_error && err == nil {
			t.Error("Error was expected but got nil. Test: %+v", test_row)
		} else if !test_row.expected_error && err != nil {
			t.Errorf("Error was not expected but got '%s'", err)
		}

		if oid_data == nil && err == nil {
			t.Error("Error was nil but got nil object")
		}

		if !test_row.expected_error {
			if oid_data.Oid.String() != test_row.expected_oid.String() {
				t.Errorf("Wrong OID. Expected: %v, got: %v", oid_data.Oid, test_row.expected_oid)
			}

			if oid_data.Type != test_row.expected_oid_type {
				t.Errorf("Wrong OID type. Expected: %v, got: %v", oid_data.Type, test_row.expected_oid_type)
			}

			if oid_data.Url != test_row.url {
				t.Errorf("Wrong url. Expected: %v, got: %v", oid_data.Url, test_row.url)
			}

			if oid_data.Jsonpath != test_row.jsonpath {
				t.Errorf("Wrong jsonpath. Expected: %v, got: %v", oid_data.Jsonpath, test_row.jsonpath)
			}
		}
	}
}

func TestStringToVariableType(t *testing.T) {
	test_data := []struct {
		t              string
		expected_type  pdu.VariableType
		expected_error bool
	}{
		{"Integer", pdu.VariableTypeInteger, false},
		{"OctetString", pdu.VariableTypeOctetString, false},
		{"ObjectIdentifier", pdu.VariableTypeObjectIdentifier, false},
		{"IpAddress", pdu.VariableTypeIPAddress, false},
		{"Counter32", pdu.VariableTypeCounter32, false},
		{"Gauge32", pdu.VariableTypeGauge32, false},
		{"TimeTicks", pdu.VariableTypeTimeTicks, false},
		{"Counter64", pdu.VariableTypeCounter64, false},
		{"INTEGER", pdu.VariableTypeNoSuchObject, true},
	}

	for _, test_row := range test_data {
		oid_type, err := string2VariableType(test_row.t)

		if test_row.expected_error && err == nil {
			t.Errorf("Error was expected for '%v' but got nil", test_row.t)
		} else if !test_row.expected_error && err != nil {
			t.Errorf("Error was not expected for '%v' but got '%s'", test_row.t, err)
		} else if !test_row.expected_error && oid_type != test_row.expected_type {
			t.Errorf("Expected type was '%v' but got '%v'", test_row.expected_type, oid_type)
		}
	}
}
