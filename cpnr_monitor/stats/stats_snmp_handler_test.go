package stats

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

type ReturnData struct {
	oid_res  value.OID
	var_type pdu.VariableType
	value    interface{}
	err      error
}

type GetTestData struct {
	oid      value.OID
	expected ReturnData
}

func check_result(expected ReturnData, actual ReturnData, t *testing.T) {
	if actual.oid_res.String() != expected.oid_res.String() {
		t.Errorf("Wrong oid res. Expected: %v. Got: %v", expected.oid_res, actual.oid_res)
	}

	if actual.var_type != expected.var_type {
		t.Errorf("Wrong var type. Expected: %v. Got: %v", expected.var_type, actual.var_type)
	}

	if fmt.Sprintf("%v", actual.value) != fmt.Sprintf("%v", expected.value) {
		t.Errorf("Wrong value. Expected: %v. Got: %v", expected.value, actual.value)
	}

	if actual.err != expected.err {
		t.Errorf("Wrong error. Expected: %v. Got: %v", expected.err, actual.err)
	}
}

func run_get_test(test_data []GetTestData, h *StatsSNMPHandler, t *testing.T) {
	for _, test_row := range test_data {
		oid_res, var_type, value, err := h.Get(test_row.oid)
		check_result(test_row.expected, ReturnData{oid_res, var_type, value, err}, t)
	}
}

func run_get_next_test(test_data []GetTestData, h *StatsSNMPHandler, t *testing.T) {
	for _, test_row := range test_data {
		oid_res, var_type, value, err := h.GetNext(test_row.oid, false, value.MustParseOID("2"))
		check_result(test_row.expected, ReturnData{oid_res, var_type, value, err}, t)
	}
}

func TestGet(t *testing.T) {
	sm := NewStatsManager()
	sm.stats["sda"] = StatsData{"sda", 3.01, 3.02, 3.03, 3.04, 3.05, 3.06, 3.07, 3.08, 3.09, 3.10, 3.11, 3.12, 3.13}
	sm.stats["dm-0"] = StatsData{"dm-0", 1.01, 1.02, 1.03, 1.04, 1.05, 1.06, 1.07, 1.08, 1.09, 1.10, 1.11, 1.12, 1.13}
	sm.stats["dm-1"] = StatsData{"dm-1", 2.01, 2.02, 2.03, 2.04, 2.05, 2.06, 2.07, 2.08, 2.09, 2.10, 2.11, 2.12, 2.13}
	sm.last_updated = time.Now()

	base_oid := "1.3.6.1.3.1"

	handler := NewStatsSNMPHandler(sm, base_oid)

	test_data := []GetTestData{
		{value.MustParseOID(base_oid + ".2.1"), ReturnData{value.MustParseOID(base_oid + ".2.1"), pdu.VariableTypeOctetString, "dm-0", nil}},
		{value.MustParseOID(base_oid + ".2.2"), ReturnData{value.MustParseOID(base_oid + ".2.2"), pdu.VariableTypeOctetString, "dm-1", nil}},
		{value.MustParseOID(base_oid + ".2.3"), ReturnData{value.MustParseOID(base_oid + ".2.3"), pdu.VariableTypeOctetString, "sda", nil}},
		{value.MustParseOID(base_oid + ".2.4"), ReturnData{nil, pdu.VariableTypeNoSuchObject, nil, nil}},
		{value.MustParseOID(base_oid + ".2.3.1"), ReturnData{nil, pdu.VariableTypeNoSuchObject, nil, nil}},
		{value.MustParseOID(base_oid + ".16.1"), ReturnData{nil, pdu.VariableTypeNoSuchObject, nil, nil}},
		{value.MustParseOID(base_oid + ".0.1"), ReturnData{nil, pdu.VariableTypeNoSuchObject, nil, nil}},
		{value.MustParseOID("1.3.6.1.3"), ReturnData{nil, pdu.VariableTypeNoSuchObject, nil, nil}},
	}

	for device := 1; device <= 3; device++ {
		for field := 1; field <= 13; field++ {
			oid := value.MustParseOID(base_oid + "." + strconv.Itoa(2+field) + "." + strconv.Itoa(device))
			value := float64(device) + float64(field)/100.0
			test_data = append(test_data, GetTestData{oid, ReturnData{oid, pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", value), nil}})
		}
	}

	run_get_test(test_data, handler, t)
}

func TestGetNext(t *testing.T) {
	sm := NewStatsManager()
	sm.stats["sda"] = StatsData{"sda", 3.01, 3.02, 3.03, 3.04, 3.05, 3.06, 3.07, 3.08, 3.09, 3.10, 3.11, 3.12, 3.13}
	sm.stats["dm-0"] = StatsData{"dm-0", 1.01, 1.02, 1.03, 1.04, 1.05, 1.06, 1.07, 1.08, 1.09, 1.10, 1.11, 1.12, 1.13}
	sm.stats["dm-1"] = StatsData{"dm-1", 2.01, 2.02, 2.03, 2.04, 2.05, 2.06, 2.07, 2.08, 2.09, 2.10, 2.11, 2.12, 2.13}
	sm.last_updated = time.Now()

	base_oid := "1.3.6.1.3.1"

	handler := NewStatsSNMPHandler(sm, base_oid)

	test_data := []GetTestData{
		{value.MustParseOID(base_oid), ReturnData{value.MustParseOID(base_oid + ".1.1"), pdu.VariableTypeInteger, 1, nil}},
		{value.MustParseOID(base_oid + ".2"), ReturnData{value.MustParseOID(base_oid + ".2.1"), pdu.VariableTypeOctetString, "dm-0", nil}},
		{value.MustParseOID(base_oid + ".2.1"), ReturnData{value.MustParseOID(base_oid + ".2.2"), pdu.VariableTypeOctetString, "dm-1", nil}},
		{value.MustParseOID(base_oid + ".2.2"), ReturnData{value.MustParseOID(base_oid + ".2.3"), pdu.VariableTypeOctetString, "sda", nil}},
	}

	run_get_next_test(test_data, handler, t)
}
