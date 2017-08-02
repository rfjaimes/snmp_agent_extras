package commons

import (
	"testing"

	"github.com/posteo/go-agentx/value"
)

func TestParseOID(t *testing.T) {
	test_data := []struct {
		oid            string
		expected       value.OID
		error_expected bool
	}{
		{"1.3.6", value.MustParseOID("1.3.6"), false},
		{".1.3.6", value.MustParseOID("1.3.6"), false},
		{".100.3.6", value.MustParseOID("100.3.6"), false},
		{".", nil, true},
		{"1.", nil, true},
	}

	for _, test_row := range test_data {
		oid, err := ParseOID(test_row.oid)
		if test_row.error_expected && err == nil {
			t.Errorf("Error was expected while parsing '%v' but got nil", test_row.oid)
		} else if !test_row.error_expected {
			if err != nil {
				t.Errorf("Error was not expected while parsing '%v' but got %v", test_row.oid, err)
			} else {
				if oid.String() != test_row.expected.String() {
					t.Errorf("Error parsing oid. Expected: '%v'\nGot %v", test_row.expected, oid)
				}
			}
		}
	}
}
