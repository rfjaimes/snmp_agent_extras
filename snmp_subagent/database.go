package snmp_subagent

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

type DBOid struct {
	BaseOid  value.OID
	Oid      value.OID
	Type     pdu.VariableType
	Url      string
	Jsonpath string
}

type DBApp struct {
	BaseOid            value.OID
	Name               string
	DiscoverUrl        string
	RediscoverInterval uint32
	Oids               map[string]DBOid
}

var db *sql.DB

func initDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if db == nil {
		return nil, errors.New("Nil DB")
	}

	return db, nil
}

func createTableApps(db *sql.DB) error {
	sql_table := `
    CREATE TABLE IF NOT EXISTS applications(
		base_oid TEXT NOT NULL PRIMARY KEY,
        name TEXT,
        discover_url TEXT,
		rediscover_interval INTEGER
    );
    `

	_, err := db.Exec(sql_table)
	return err
}

func createTableOids(db *sql.DB) error {
	sql_table := `
    CREATE TABLE IF NOT EXISTS oids(
		base_oid TEXT NOT NULL,
		oid TEXT NOT NULL,
		type TEXT NOT NULL,
        url TEXT NOT NULL,
        jsonpath TEXT NOT NULL,
		PRIMARY KEY(base_oid, oid)
    );
    `

	_, err := db.Exec(sql_table)
	return err
}

func loadFromDB(db *sql.DB) (map[string]DBApp, error) {

	if db == nil {
		return nil, errors.New("Nil db")
	}
	sql_readall := `
	SELECT app.base_oid, app.name, app.discover_url, app.rediscover_interval,
		   oid.oid, oid.type, oid.url, oid.jsonpath
	FROM applications app, oids oid
	WHERE app.base_oid = oid.base_oid
	ORDER BY app.base_oid;`

	rows, err := db.Query(sql_readall)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var db_base_oid string
	var db_name string
	var db_discover_url string
	var db_rediscover_interval uint32
	var db_oid string
	var db_oid_type string
	var db_url string
	var db_jsonpath string

	var base_oid value.OID
	var oid value.OID
	var oid_type pdu.VariableType

	result := make(map[string]DBApp)
	var app DBApp
	var app_oid DBOid
	var app_oids map[string]DBOid

	for rows.Next() {
		if err = rows.Scan(&db_base_oid, &db_name, &db_discover_url, &db_rediscover_interval, &db_oid, &db_oid_type, &db_url, &db_jsonpath); err != nil {
			return nil, err
		}

		if base_oid, err = ParseOID(db_base_oid); err != nil {
			log.Warningf("Wrong oid format for '%v': %v", db_base_oid, err)
			continue
		}

		if oid, err = ParseOID(db_oid); err != nil {
			log.Warningf("Wrong oid format for '%v': %v", db_oid, err)
			continue
		}

		if oid_type, err = string2VariableType(db_oid_type); err != nil {
			log.Warningf("OID '%v' from DB has wrong oid type (%v). OID will not be registered.", oid, db_oid_type)
			continue
		}

		app_oid = DBOid{
			BaseOid:  base_oid,
			Oid:      oid,
			Type:     oid_type,
			Url:      db_url,
			Jsonpath: db_jsonpath,
		}

		var ok bool
		if app, ok = result[base_oid.String()]; !ok {
			app = DBApp{
				BaseOid:            base_oid,
				Name:               db_name,
				DiscoverUrl:        db_discover_url,
				RediscoverInterval: db_rediscover_interval,
				Oids:               make(map[string]DBOid),
			}
		}
		app_oids = app.Oids

		app_oids[oid.String()] = app_oid
		app.Oids = app_oids
		result[base_oid.String()] = app
	}

	return result, nil
}

func saveApplication(db *sql.DB, app *Application) error {
	sql_add_app := `
    INSERT OR REPLACE INTO applications(
		base_oid,
        name,
        discover_url,
		rediscover_interval
    ) values (?, ?, ?, ?)
    `

	sql_delete_oids := "DELETE FROM oids WHERE base_oid = ?"

	sql_add_oid := `
    INSERT OR REPLACE INTO oids(
		base_oid,
		oid,
		type,
        url,
        jsonpath
    ) values (?, ?, ?, ?, ?)
    `

	stmt_add_app, err := db.Prepare(sql_add_app)
	if err != nil {
		return err
	}

	stmt_del_oids, err := db.Prepare(sql_delete_oids)
	if err != nil {
		return err
	}

	stmt_add_oid, err := db.Prepare(sql_add_oid)
	if err != nil {
		return err
	}

	defer stmt_add_app.Close()
	defer stmt_del_oids.Close()
	defer stmt_add_oid.Close()

	if _, err2 := stmt_add_app.Exec(app.base_oid.String(), app.name, app.discover_url, app.discover_interval); err2 != nil {
		return err2
	}

	if _, err2 := stmt_del_oids.Exec(app.base_oid.String()); err2 != nil {
		return err2
	}

	for oid, oid_data := range app.oids {
		t, err := variableType2String(oid_data.Type)
		if err != nil {
			return err
		}
		if _, err2 := stmt_add_oid.Exec(app.base_oid.String(), oid, t, oid_data.Url, oid_data.Jsonpath); err2 != nil {
			return err2
		}

	}

	return nil

}

func deleteApplication(db *sql.DB, base_oid value.OID) error {
	sql_delete_app := "DELETE FROM applications WHERE base_oid = ?"
	sql_delete_oids := "DELETE FROM oids WHERE base_oid = ?"

	stmt_delete_app, err := db.Prepare(sql_delete_app)
	if err != nil {
		return err
	}

	stmt_delete_oids, err := db.Prepare(sql_delete_oids)
	if err != nil {
		return err
	}

	defer stmt_delete_app.Close()
	defer stmt_delete_oids.Close()

	if _, err2 := stmt_delete_app.Exec(base_oid.String()); err2 != nil {
		return err2
	}

	if _, err2 := stmt_delete_oids.Exec(base_oid.String()); err2 != nil {
		return err2
	}

	return nil
}
