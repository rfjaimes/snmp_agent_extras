package snmp_subagent

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

var base_oid1 = value.MustParseOID("1.3.6.1.4.1.8072.100.1")
var base_oid2 = value.MustParseOID("1.3.6.1.4.1.8072.100.2")

var db_app1 = DBApp{
	Name:               "App 1",
	BaseOid:            base_oid1,
	DiscoverUrl:        "",
	RediscoverInterval: 0,
	Oids: map[string]DBOid{
		"1.3.6.1.4.1.8072.100.1.1.1.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.1.0"), Type: pdu.VariableTypeOctetString, Url: "http://app1-mock/aaa", Jsonpath: "$.first"},
		"1.3.6.1.4.1.8072.100.1.1.2.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.2.0"), Type: pdu.VariableTypeOctetString, Url: "http://app1-mock/aaa", Jsonpath: "$.second"},
		"1.3.6.1.4.1.8072.100.1.1.3.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.3.0"), Type: pdu.VariableTypeInteger, Url: "http://app1-mock/aaa", Jsonpath: "$.third"},
		"1.3.6.1.4.1.8072.100.1.1.4.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.4.0"), Type: pdu.VariableTypeTimeTicks, Url: "http://app1-mock/aaa", Jsonpath: "$.uptime"},
		"1.3.6.1.4.1.8072.100.1.1.5.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.5.0"), Type: pdu.VariableTypeIPAddress, Url: "http://app1-mock/bbb", Jsonpath: "$.first"},
		"1.3.6.1.4.1.8072.100.1.1.6.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.6.0"), Type: pdu.VariableTypeCounter32, Url: "http://app1-mock/bbb", Jsonpath: "$.second"},
		"1.3.6.1.4.1.8072.100.1.1.7.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.7.0"), Type: pdu.VariableTypeGauge32, Url: "http://app1-mock/bbb", Jsonpath: "$.third"},
		"1.3.6.1.4.1.8072.100.1.1.8.0": DBOid{BaseOid: base_oid1, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.1.1.8.0"), Type: pdu.VariableTypeCounter64, Url: "http://app1-mock/bbb", Jsonpath: `$["aaa.bbb.ccc"]`},
	},
}

var db_app2 = DBApp{
	Name:               "App 2",
	BaseOid:            base_oid2,
	DiscoverUrl:        "",
	RediscoverInterval: 0,
	Oids: map[string]DBOid{
		"1.3.6.1.4.1.8072.100.2.1.1.0": DBOid{BaseOid: base_oid2, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.2.1.1.0"), Type: pdu.VariableTypeOctetString, Url: "http://app2-mock/data", Jsonpath: "$.name"},
		"1.3.6.1.4.1.8072.100.2.1.2.0": DBOid{BaseOid: base_oid2, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.2.1.2.0"), Type: pdu.VariableTypeOctetString, Url: "http://app2-mock/data", Jsonpath: "$.version"},
		"1.3.6.1.4.1.8072.100.2.1.3.0": DBOid{BaseOid: base_oid2, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.2.1.3.0"), Type: pdu.VariableTypeInteger, Url: "http://app2-mock/data", Jsonpath: "$.status"},
		"1.3.6.1.4.1.8072.100.2.1.4.0": DBOid{BaseOid: base_oid2, Oid: value.MustParseOID("1.3.6.1.4.1.8072.100.2.1.4.0"), Type: pdu.VariableTypeTimeTicks, Url: "http://app2-mock/data", Jsonpath: "$.uptime"},
	},
}

var expected = map[string]DBApp{
	"1.3.6.1.4.1.8072.100.1": db_app1,
	"1.3.6.1.4.1.8072.100.2": db_app2,
}

func TestLoadFromDB(t *testing.T) {
	db, err := initDB("../test/db/good.db")
	if err != nil {
		t.Errorf("Error initing db: %v", err)
	}

	apps, err := loadFromDB(db)
	if err != nil {
		t.Errorf("Error loading db: %v", err)
	}

	if apps == nil {
		t.Error("Error loading db. Apps is nil")
	}

	if !reflect.DeepEqual(apps, expected) {
		t.Errorf("Error loading apps.\nExp: %v\nGot: %v", expected, apps)
	}
}

func TestNewSaveAndLoadFromDB(t *testing.T) {
	db_file := "../test/db/tmp_new_db.db"

	db, err := initDB(db_file)
	if err != nil {
		t.Errorf("Error initing db: %v", err)
	}
	defer os.Remove(db_file)

	if err = createTableApps(db); err != nil {
		t.Errorf("Error creating table: %v", err)
	}

	if err = createTableOids(db); err != nil {
		t.Errorf("Error creating table: %v", err)
	}

	oid1_1, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.1.0", "OctetString", "http://app1-mock/aaa", "$.first")
	oid1_2, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.2.0", "OctetString", "http://app1-mock/aaa", "$.second")
	oid1_3, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.3.0", "Integer", "http://app1-mock/aaa", "$.third")
	oid1_4, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.4.0", "TimeTicks", "http://app1-mock/aaa", "$.uptime")
	oid1_5, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.5.0", "IpAddress", "http://app1-mock/bbb", "$.first")
	oid1_6, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.6.0", "Counter32", "http://app1-mock/bbb", "$.second")
	oid1_7, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.7.0", "Gauge32", "http://app1-mock/bbb", "$.third")
	oid1_8, _ := NewOIDData("1.3.6.1.4.1.8072.100.1.1.8.0", "Counter64", "http://app1-mock/bbb", `$["aaa.bbb.ccc"]`)

	oid2_1, _ := NewOIDData("1.3.6.1.4.1.8072.100.2.1.1.0", "OctetString", "http://app2-mock/data", "$.name")
	oid2_2, _ := NewOIDData("1.3.6.1.4.1.8072.100.2.1.2.0", "OctetString", "http://app2-mock/data", "$.version")
	oid2_3, _ := NewOIDData("1.3.6.1.4.1.8072.100.2.1.3.0", "Integer", "http://app2-mock/data", "$.status")
	oid2_4, _ := NewOIDData("1.3.6.1.4.1.8072.100.2.1.4.0", "TimeTicks", "http://app2-mock/data", "$.uptime")

	oids_app1 := map[string]*OIDData{
		"1.3.6.1.4.1.8072.100.1.1.1.0": oid1_1,
		"1.3.6.1.4.1.8072.100.1.1.2.0": oid1_2,
		"1.3.6.1.4.1.8072.100.1.1.3.0": oid1_3,
		"1.3.6.1.4.1.8072.100.1.1.4.0": oid1_4,
		"1.3.6.1.4.1.8072.100.1.1.5.0": oid1_5,
		"1.3.6.1.4.1.8072.100.1.1.6.0": oid1_6,
		"1.3.6.1.4.1.8072.100.1.1.7.0": oid1_7,
		"1.3.6.1.4.1.8072.100.1.1.8.0": oid1_8,
	}

	oids_app2 := map[string]*OIDData{
		"1.3.6.1.4.1.8072.100.2.1.1.0": oid2_1,
		"1.3.6.1.4.1.8072.100.2.1.2.0": oid2_2,
		"1.3.6.1.4.1.8072.100.2.1.3.0": oid2_3,
		"1.3.6.1.4.1.8072.100.2.1.4.0": oid2_4,
	}

	app1 := NewApplication("App 1", "", value.MustParseOID("1.3.6.1.4.1.8072.100.1"), 0, 100, 100, oids_app1)
	app2 := NewApplication("App 2", "", value.MustParseOID("1.3.6.1.4.1.8072.100.2"), 0, 100, 100, oids_app2)

	if err = saveApplication(db, app1); err != nil {
		t.Errorf("Error saving application 1: %v", err)
	}

	if err = saveApplication(db, app2); err != nil {
		t.Errorf("Error saving application 2: %v", err)
	}

	apps, err := loadFromDB(db)
	if err != nil {
		t.Errorf("Error loading db: %v", err)
	}

	if apps == nil {
		t.Error("Error loading db. Apps is nil")
	}

	if !reflect.DeepEqual(apps, expected) {
		t.Errorf("Error loading apps.\nExp: %v\nGot: %v", expected, apps)
	}
}

func TestDeleteFromDB(t *testing.T) {
	in_db_file := "../test/db/good.db"
	db_file := "../test/db/tmp_delete_test.db"
	// Copy db file to do the delete test
	if data, err := ioutil.ReadFile(in_db_file); err != nil {
		t.Errorf("Error reading %v: %v", in_db_file, err)
	} else {
		if err = ioutil.WriteFile(db_file, data, 0644); err != nil {
			t.Errorf("Error writing %v: %v", db_file, err)
		}
	}

	db, err := initDB(db_file)
	if err != nil {
		t.Errorf("Error initing db: %v", err)
	}
	defer os.Remove(db_file)

	deleteApplication(db, base_oid1)

	apps, err := loadFromDB(db)
	if err != nil {
		t.Errorf("Error loading db: %v", err)
	}

	if apps == nil {
		t.Error("Error loading db. Apps is nil")
	}

	expected = map[string]DBApp{
		"1.3.6.1.4.1.8072.100.2": db_app2,
	}

	if !reflect.DeepEqual(apps, expected) {
		t.Errorf("Error loading apps.\nExp: %v\nGot: %v", expected, apps)
	}
}
