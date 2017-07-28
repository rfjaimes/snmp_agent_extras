package snmp_subagent

import (
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

func app1_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch r.URL.String() {
	case "/discover":
		w.Write([]byte(`{ "status": "success", "errcode": "", "errmessage": ""  , "_embedded": { "oids": [
		{"oid": "1.3.6.1.4.1.8072.100.1.1.1.0", "type": "OctetString", "url": "/aaa", "jsonpath": "$.first"},
{"oid": ".1.3.6.1.4.1.8072.100.1.1.2.0", "type": "OctetString", "url": "/aaa", "jsonpath": "$.second"},
{"oid": "1.3.6.1.4.1.8072.100.1.1.3.0", "type": "Integer", "url": "/aaa", "jsonpath": "$.third"},
{"oid": "1.3.6.1.4.1.8072.100.1.1.4.0", "type": "TimeTicks", "url": "/aaa", "jsonpath": "$.uptime"},
{"oid": "1.3.6.1.4.1.8072.100.1.1.5.0", "type": "IpAddress", "url": "/bbb", "jsonpath": "$.first"},
{"oid": "1.3.6.1.4.1.8072.100.1.1.6.0", "type": "Counter32", "url": "/bbb", "jsonpath": "$.second"},
{"oid": "1.3.6.1.4.1.8072.100.1.1.7.0", "type": "Gauge32", "url": "/bbb", "jsonpath": "$.third"},
{"oid": "1.3.6.1.4.1.8072.100.1.1.8.0", "type": "Counter64", "url": "/bbb", "jsonpath": "$[\"aaa.bbb.ccc\"]"}
 ] } }`))
	case "/aaa":
		w.Write([]byte(`{
			"first": "App 1",
			"second": "1.5.12",
			"third": 1,
			"uptime": 123456
		}`))
	case "/bbb":
		w.Write([]byte(`{
			"first": "10.20.30.40",
			"second": 4222,
			"third": "11333",
			"aaa.bbb.ccc": 771
		}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func app2_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch r.URL.String() {
	case "/discover":
		w.Write([]byte(`{ "status": "success", "errcode": "", "errmessage": ""  , "_embedded": { "oids": [
		{"oid": "1.3.6.1.4.1.8072.100.2.1.1.0", "type": "OctetString", "url": "/pepe", "jsonpath": "$.laaa"},
{"oid": "1.3.6.1.4.1.8072.100.2.1.2.0", "type": "OctetString", "url": "/pepe", "jsonpath": "$.leee"}
 ] } }`))
	case "/pepe":
		w.Write([]byte(`{
			"laaa": "App 2",
			"leee": "1.5.13"
		}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func problem_app1_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch r.URL.String() {
	case "/abc":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(`{"aaa": "123"}`))
	case "/slow":
		time.Sleep(400 * time.Millisecond)
		w.Write([]byte(`{"aaa", "123"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestParseDiscovery(t *testing.T) {
	db1 := `{ "status": "success", "errcode": "", "errmessage": ""  , "_embedded": { "oids": [{"oid": "1.3.6.1.2.3.5.1.0", "type": "INTEGER", "url": "http://abc", "jsonpath": "$.first"}, {"oid": "1.3.6.1.2.3.5.2.0", "type": "INTEGER", "url": "http://abc", "jsonpath": "$.second"}, {"oid": "1.3.6.1.2.3.5.3.0", "type": "INTEGER", "url": "http://abc", "jsonpath": "$.third"}] } }`
	exp_1 := DiscoveryGet{
		Status:  "success",
		ErrCode: "",
		ErrMsg:  "",
		Embedded: EmbeddedDiscovery{
			Oids: []OIDResource{
				OIDResource{Oid: "1.3.6.1.2.3.5.1.0", Type: "INTEGER", Url: "http://abc", Jsonpath: "$.first"},
				OIDResource{Oid: "1.3.6.1.2.3.5.2.0", Type: "INTEGER", Url: "http://abc", Jsonpath: "$.second"},
				OIDResource{Oid: "1.3.6.1.2.3.5.3.0", Type: "INTEGER", Url: "http://abc", Jsonpath: "$.third"},
			},
		},
	}

	test_data := []struct {
		body              []byte
		expected_resource DiscoveryGet
		error_expected    bool
	}{
		{[]byte(db1), exp_1, false},
	}

	for _, test_row := range test_data {
		resource, err := ParseDiscoveryResponse(test_row.body)
		if err == nil && test_row.error_expected {
			t.Errorf("Error was expected while parsing %s", test_row.body)
		} else if err != nil && !test_row.error_expected {
			t.Errorf("Error was not expected while parsing %s. Err: %v", test_row.body, err)
		} else if err == nil && !reflect.DeepEqual(test_row.expected_resource, resource) {
			t.Errorf("Error parsing discovery.\nExpected:%+v\nGot:%+v", test_row.expected_resource, resource)
		}

	}
}

func TestDiscover(t *testing.T) {
	app_server := httptest.NewServer(http.HandlerFunc(app1_handler))

	defer app_server.Close()

	base_oid := value.MustParseOID("1.3.6")

	url := app_server.URL + "/discover"

	app := NewApplication("App 1", url, base_oid, 100, 100, 100, nil)

	app.discover()

	expected := map[string]*OIDData{
		"1.3.6.1.4.1.8072.100.1.1.1.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.1.0"), Type: pdu.VariableTypeOctetString, Url: "/aaa", Jsonpath: `$.first`},
		"1.3.6.1.4.1.8072.100.1.1.2.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.2.0"), Type: pdu.VariableTypeOctetString, Url: "/aaa", Jsonpath: `$.second`},
		"1.3.6.1.4.1.8072.100.1.1.3.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.3.0"), Type: pdu.VariableTypeInteger, Url: "/aaa", Jsonpath: `$.third`},
		"1.3.6.1.4.1.8072.100.1.1.4.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.4.0"), Type: pdu.VariableTypeTimeTicks, Url: "/aaa", Jsonpath: `$.uptime`},
		"1.3.6.1.4.1.8072.100.1.1.5.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.5.0"), Type: pdu.VariableTypeIPAddress, Url: "/bbb", Jsonpath: `$.first`},
		"1.3.6.1.4.1.8072.100.1.1.6.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.6.0"), Type: pdu.VariableTypeCounter32, Url: "/bbb", Jsonpath: `$.second`},
		"1.3.6.1.4.1.8072.100.1.1.7.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.7.0"), Type: pdu.VariableTypeGauge32, Url: "/bbb", Jsonpath: `$.third`},
		"1.3.6.1.4.1.8072.100.1.1.8.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.8.0"), Type: pdu.VariableTypeCounter64, Url: "/bbb", Jsonpath: `$["aaa.bbb.ccc"]`},
	}

	expected_keys := make([]string, len(expected))
	i := 0
	for k, _ := range expected {
		expected_keys[i] = k
		i++
	}
	sort.Strings(expected_keys)

	got_keys := make([]string, len(app.oids))
	i = 0
	for k, _ := range app.oids {
		got_keys[i] = k
		i++
	}
	sort.Strings(got_keys)

	if !reflect.DeepEqual(expected_keys, got_keys) {
		t.Errorf("Error loading oids.\nExpected: %+v\nGot: %+v", expected_keys, got_keys)
	}

	for _, key := range expected_keys {
		exp := expected[key]
		got := app.oids[key]

		if !reflect.DeepEqual(exp.Oid, got.Oid) {
			t.Errorf("Wrong OID.\nExpected: %+v\nGot: %+v", exp.Oid, got.Oid)
		}
		if exp.Type != got.Type {
			t.Errorf("Wrong type.\nExpected: %+v\nGot: %+v", exp.Type, got.Type)
		}
		if exp.Url != got.Url {
			t.Errorf("Wrong url.\nExpected: %+v\nGot: %+v", exp.Url, got.Url)
		}
		if exp.Jsonpath != got.Jsonpath {
			t.Errorf("Wrong jsonpath.\nExpected: %+v\nGot: %+v", exp.Jsonpath, got.Jsonpath)
		}

	}
}

func TestDiscoverReplace(t *testing.T) {
	app1_server := httptest.NewServer(http.HandlerFunc(app1_handler))
	app2_server := httptest.NewServer(http.HandlerFunc(app2_handler))

	defer app1_server.Close()
	defer app2_server.Close()

	url1 := app1_server.URL + "/discover"
	url2 := app2_server.URL + "/discover"

	base_oid := value.MustParseOID("1.3.6")

	app := NewApplication("App 1", url1, base_oid, 100, 100, 100, nil)
	app.discover()
	app.discover_url = url2
	app.discover()

	expected := map[string]*OIDData{
		"1.3.6.1.4.1.8072.100.2.1.1.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.2.1.1.0"), Type: pdu.VariableTypeOctetString, Url: "/pepe", Jsonpath: `$.laaa`},
		"1.3.6.1.4.1.8072.100.2.1.2.0": {Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.2.1.2.0"), Type: pdu.VariableTypeOctetString, Url: "/pepe", Jsonpath: `$.leee`},
	}

	expected_keys := make([]string, len(expected))
	i := 0
	for k, _ := range expected {
		expected_keys[i] = k
		i++
	}
	sort.Strings(expected_keys)

	got_keys := make([]string, len(app.oids))
	i = 0
	for k, _ := range app.oids {
		got_keys[i] = k
		i++
	}
	sort.Strings(got_keys)

	if !reflect.DeepEqual(expected_keys, got_keys) {
		t.Errorf("Error loading oids.\nExpected: %+v\nGot: %+v", expected_keys, got_keys)
	}

	for _, key := range expected_keys {
		exp := expected[key]
		got := app.oids[key]

		if !reflect.DeepEqual(exp.Oid, got.Oid) {
			t.Errorf("Wrong OID.\nExpected: %+v\nGot: %+v", exp.Oid, got.Oid)
		}
		if exp.Type != got.Type {
			t.Errorf("Wrong type.\nExpected: %+v\nGot: %+v", exp.Type, got.Type)
		}
		if exp.Url != got.Url {
			t.Errorf("Wrong url.\nExpected: %+v\nGot: %+v", exp.Url, got.Url)
		}
		if exp.Jsonpath != got.Jsonpath {
			t.Errorf("Wrong jsonpath.\nExpected: %+v\nGot: %+v", exp.Jsonpath, got.Jsonpath)
		}

	}
}

func TestProcessOidResponse(t *testing.T) {
	response_aaa := []byte(`{
			"first": "App 1",
			"second": "1.5.12",
			"third": 1,
			"uptime": 123456,
			"someoid": "1.3.6.101"
		}`)
	response_bbb := []byte(`{
			"first": "10.20.30.40",
			"second": 4222,
			"third": "11333",
			"aaa.bbb.ccc": 771
		}`)

	oid_data_string, _ := NewOIDData("1.3.100", "OctetString", "aaa", "$.first")
	oid_data_integer, _ := NewOIDData("1.3.100", "Integer", "aaa", "$.third")
	oid_data_time, _ := NewOIDData("1.3.100", "TimeTicks", "aaa", "$.uptime")
	oid_data_oid, _ := NewOIDData("1.3.100", "ObjectIdentifier", "aaa", "$.someoid")
	oid_data_ip, _ := NewOIDData("1.3.100", "IpAddress", "bbb", "$.first")
	oid_data_counter32, _ := NewOIDData("1.3.100", "Counter32", "bbb", "$.second")
	oid_data_gauge32, _ := NewOIDData("1.3.100", "Gauge32", "bbb", "$.third")
	oid_data_counter64, _ := NewOIDData("1.3.100", "Counter64", "bbb", `$["aaa.bbb.ccc"]`)

	test_data := []struct {
		oid_body       []byte
		oid_data       *OIDData
		exp_value      interface{}
		error_expected bool
	}{
		{response_aaa, oid_data_string, "App 1", false},
		{response_aaa, oid_data_integer, int32(1), false},
		{response_aaa, oid_data_time, time.Duration(123456 * time.Second), false},
		{response_aaa, oid_data_oid, "1.3.6.101", false},
		{response_bbb, oid_data_ip, net.IPv4(10, 20, 30, 40).To4(), false},
		{response_bbb, oid_data_counter32, uint32(4222), false},
		{response_bbb, oid_data_gauge32, uint32(11333), false},
		{response_bbb, oid_data_counter64, uint64(771), false},

		{[]byte(`{"first": "ABC DEF"}`), oid_data_string, "ABC DEF", false},
		{[]byte(`{"first": 10}`), oid_data_string, nil, true},
		{[]byte(`{"aaa": "aaa"}`), oid_data_string, nil, true},

		{[]byte(`{"third": "123"}`), oid_data_integer, int32(123), false},
		{[]byte(`{"third": "-123"}`), oid_data_integer, int32(-123), false},
		{[]byte(`{"third": -123}`), oid_data_integer, int32(-123), false},
		{[]byte(`{"third": "10a"}`), oid_data_integer, nil, true},

		{[]byte(`{"someoid": "1.3.6.100.4"}`), oid_data_oid, "1.3.6.100.4", false},
		{[]byte(`{"someoid": "1.3.6.100.4.A"}`), oid_data_oid, nil, true},
		{[]byte(`{"someoid": 10}`), oid_data_oid, nil, true},

		{[]byte(`{"first": "1.2.3.4"}`), oid_data_ip, net.IPv4(1, 2, 3, 4).To4(), false},
		{[]byte(`{"first": "10.200.215.14"}`), oid_data_ip, net.IPv4(10, 200, 215, 14).To4(), false},
		{[]byte(`{"first": "1.2.3"}`), oid_data_ip, nil, true},
		{[]byte(`{"first": "1.2.3."}`), oid_data_ip, nil, true},
		{[]byte(`{"first": "1.2.3.256"}`), oid_data_ip, nil, true},
		{[]byte(`{"first": "a.b.c.d"}`), oid_data_ip, nil, true},

		{[]byte(`{"second": "10"}`), oid_data_counter32, uint32(10), false},
		{[]byte(`{"second": 10}`), oid_data_counter32, uint32(10), false},
		{[]byte(`{"second": -10}`), oid_data_counter32, nil, true},
		{[]byte(`{"second": 4294967296}`), oid_data_counter32, nil, true},
		{[]byte(`{"second": "10a"}`), oid_data_counter32, nil, true},

		{[]byte(`{"third": "10"}`), oid_data_gauge32, uint32(10), false},
		{[]byte(`{"third": 10}`), oid_data_gauge32, uint32(10), false},
		{[]byte(`{"third": -10}`), oid_data_gauge32, nil, true},
		{[]byte(`{"third": 4294967296}`), oid_data_gauge32, nil, true},
		{[]byte(`{"third": "10a"}`), oid_data_gauge32, nil, true},

		{[]byte(`{"uptime": "10"}`), oid_data_time, time.Duration(10 * time.Second), false},
		{[]byte(`{"uptime": 10}`), oid_data_time, time.Duration(10 * time.Second), false},
		{[]byte(`{"uptime": 4294967295}`), oid_data_time, time.Duration(4294967295 * time.Second), false},
		{[]byte(`{"uptime": -10}`), oid_data_time, nil, true},
		{[]byte(`{"uptime": "tomorrow"}`), oid_data_time, nil, true},

		{[]byte(`{"aaa.bbb.ccc": "10"}`), oid_data_counter64, uint64(10), false},
		{[]byte(`{"aaa.bbb.ccc": 10}`), oid_data_counter64, uint64(10), false},
		{[]byte(`{"aaa.bbb.ccc": 4294967296}`), oid_data_counter64, uint64(4294967296), false},
		{[]byte(`{"aaa.bbb.ccc": -10}`), oid_data_counter64, nil, true},
		{[]byte(`{"aaa.bbb.ccc": "10a"}`), oid_data_counter64, nil, true},

		// TODO: Opaque
	}

	for _, test_row := range test_data {
		r_value, err := processOidResponse(test_row.oid_body, test_row.oid_data)

		if err == nil && test_row.error_expected {
			t.Errorf("Error was expected while processing %s", string(test_row.oid_body))
		} else if err != nil && !test_row.error_expected {
			t.Errorf("Error was not expected while processing %s. Err: %v", string(test_row.oid_body), err)
		} else if err == nil {
			if test_row.oid_data.Type == pdu.VariableTypeOpaque ||
				test_row.oid_data.Type == pdu.VariableTypeIPAddress {
				if !reflect.DeepEqual(test_row.exp_value, r_value) {
					t.Errorf("Value mismatch. Expected: '%v'. Got: '%v'", test_row.exp_value, r_value)
				}
			} else {
				if r_value != test_row.exp_value {
					t.Errorf("Value mismatch. Expected: '%v'. Got: '%v'", test_row.exp_value, r_value)
				}
			}
		}
	}
}

func TestGet(t *testing.T) {
	app_server := httptest.NewServer(http.HandlerFunc(app1_handler))

	defer app_server.Close()

	base_oid := value.MustParseOID("1.3.6")

	url1 := app_server.URL + "/aaa"
	url2 := app_server.URL + "/bbb"

	oid1, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.1.0", "OctetString", url1, `$.first`)
	oid2, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.2.0", "OctetString", url1, `$.second`)
	oid3, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.3.0", "Integer", url1, `$.third`)
	oid4, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.4.0", "TimeTicks", url1, `$.uptime`)
	oid5, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.5.0", "IpAddress", url2, `$.first`)
	oid6, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.6.0", "Counter32", url2, `$.second`)
	oid7, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.7.0", "Gauge32", url2, `$.third`)
	oid8, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.8.0", "Counter64", url2, `$["aaa.bbb.ccc"]`)

	oids := map[string]*OIDData{
		"1.3.6.1.4.1.8072.100.1.1.1.0": oid1,
		"1.3.6.1.4.1.8072.100.1.1.2.0": oid2,
		"1.3.6.1.4.1.8072.100.1.1.3.0": oid3,
		"1.3.6.1.4.1.8072.100.1.1.4.0": oid4,
		"1.3.6.1.4.1.8072.100.1.1.5.0": oid5,
		"1.3.6.1.4.1.8072.100.1.1.6.0": oid6,
		"1.3.6.1.4.1.8072.100.1.1.7.0": oid7,
		"1.3.6.1.4.1.8072.100.1.1.8.0": oid8,
	}

	app := NewApplication("App 1", app_server.URL, base_oid, 100, 100, 100, oids)

	test_data := []struct {
		oid   value.OID
		value interface{}
	}{
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.1.0"), "App 1"},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.2.0"), "1.5.12"},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.3.0"), int32(1)},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.4.0"), time.Duration(123456 * time.Second)},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.5.0"), net.IPv4(10, 20, 30, 40).To4()},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.6.0"), uint32(4222)},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.7.0"), uint32(11333)},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.8.0"), uint64(771)},
	}

	ip_type := reflect.TypeOf(net.IPv4(0, 0, 0, 0).To4())

	for _, test_row := range test_data {
		r_value := app.Get(test_row.oid)
		switch reflect.TypeOf(r_value) {
		case ip_type:
			if !reflect.DeepEqual(test_row.value, r_value) {
				t.Errorf("Value mismatch. Expected: '%v'. Got: '%v'", test_row.value, r_value)
			}
		default:
			if test_row.value != r_value {
				t.Errorf("Value mismatch. Expected: '%v'. Got: '%v'", test_row.value, r_value)
			}
		}
	}
}

/*

func TestGetErrors(t *testing.T) {
	TODO
	app_server := httptest.NewServer(http.HandlerFunc(problem_app1_handler))

	defer app_server.Close()

	base_oid := value.MustParseOID("1.3.6")

	app := NewApplication("App 1", app_server.URL, base_oid, 100, 100)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.1.0", pdu.VariableTypeOctetString)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.2.0", pdu.VariableTypeOctetString)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.3.0", pdu.VariableTypeInteger)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.4.0", pdu.VariableTypeTimeTicks)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.5.0", pdu.VariableTypeOctetString)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.6.0", pdu.VariableTypeOctetString)
	app.AddOid("1.3.6.1.4.1.8072.100.3.1.7.0", pdu.VariableTypeOctetString)

	test_data := []struct {
		oid   value.OID
		value interface{}
	}{
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.1.0"), nil},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.2.0"), nil},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.3.0"), nil},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.4.0"), nil},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.5.0"), nil},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.6.0"), nil},
		{value.MustParseOID("1.3.6.1.4.1.8072.100.3.1.7.0"), nil},
	}

	for _, test_row := range test_data {
		r_value := app.Get(test_row.oid)
		if test_row.value != r_value {
			t.Errorf("Value mismatch. Expected: '%v'. Got: '%v'", test_row.value, r_value)
		}
	}
}
*/
