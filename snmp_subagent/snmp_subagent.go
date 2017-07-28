package snmp_subagent

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	logging "github.com/op/go-logging"
	agentx "github.com/posteo/go-agentx"
	"github.com/posteo/go-agentx/value"
)

var log = logging.MustGetLogger("sentinel-snmp-subagent")

type SNMPSubagent struct {
	applications map[string]*Application
	m            *sync.Mutex
	config       Config
}

func NewSNMPSubagent() *SNMPSubagent {
	s := &SNMPSubagent{
		applications: make(map[string]*Application),
		m:            &sync.Mutex{},
		config:       DefaultConfig(),
	}

	return s
}

func (self *SNMPSubagent) Initialize(config Config) error {
	self.config = config
	if err := self.initDB(); err != nil {
		return err
	}

	// Load oids and applications
	apps, err := loadFromDB(db)
	if err != nil {
		return err
	}

	for _, app := range apps {
		oids := map[string]*OIDData{}
		for _, db_oid := range app.Oids {
			oid_s := db_oid.Oid.String()
			type_s, _ := variableType2String(db_oid.Type)
			oid, err := NewOIDData(oid_s, type_s, db_oid.Url, db_oid.Jsonpath)
			if err != nil {
				log.Errorf("Error loading oid %v: %v", oid_s, err)
				continue
			}
			oids[oid_s] = oid
		}

		self.appRegister(app.Name, app.DiscoverUrl, app.BaseOid, app.RediscoverInterval, oids)
	}

	return nil
}

func (self *SNMPSubagent) Register(name string, discover_url string, base_oid value.OID, discover_interval uint32, oids map[string]*OIDData) error {
	application, err := self.appRegister(name, discover_url, base_oid, discover_interval, oids)
	if err != nil {
		return err
	}

	self.m.Lock()
	defer self.m.Unlock()
	if err = saveApplication(db, application); err != nil {
		log.Errorf("Couldn't save application in db: %v", err)
		return err
	}

	return nil
}

func (self *SNMPSubagent) appRegister(name string, discover_url string, base_oid value.OID, discover_interval uint32, oids map[string]*OIDData) (*Application, error) {
	self.m.Lock()
	application, ok := self.applications[base_oid.String()]
	self.m.Unlock()
	if ok {
		self.Unregister(base_oid)
	}

	application = NewApplication(name, discover_url, base_oid, discover_interval, self.config.GetTimeout, self.config.DiscoverTimeout, oids)

	self.m.Lock()
	defer self.m.Unlock()

	client := &agentx.Client{
		Net:               self.config.AgentXProtocol,
		Address:           self.config.AgentXAddress,
		Timeout:           1 * time.Minute,
		ReconnectInterval: 1 * time.Second,
	}

	if err := client.Open(); err != nil {
		return nil, err
	}

	application.client = client

	session, err := client.Session()
	if err != nil {
		return nil, err
	}

	session.Handler = application.handler
	application.session = session

	if err := session.Register(127, base_oid); err != nil {
		return nil, err
	}

	self.applications[base_oid.String()] = application
	log.Info(base_oid, "successfully registered")

	if discover_url != "" {
		application.discover()

		if discover_interval > 0 {
			go application.rediscover()
		}
	}

	return application, nil
}

func (self *SNMPSubagent) Unregister(base_oid value.OID) {
	self.m.Lock()
	defer self.m.Unlock()
	application, ok := self.applications[base_oid.String()]
	if !ok {
		log.Infof("Unregistration for '%v' failed because it was not registered", base_oid)
		return
	}

	application.stopRediscover()

	err := application.session.Unregister(127, base_oid)
	if err != nil {
		log.Warningf("Error trying to unregister App '%v': %v", application.name, err)
		return
	}

	err = application.session.Close()
	if err != nil {
		log.Warningf("Error closing session: %v", err)
		return
	}

	err = application.client.Close()
	if err != nil {
		log.Warningf("Error trying to close client: %v", err)
		return
	}

	delete(self.applications, base_oid.String())
	// Delete from DB
	if err = deleteApplication(db, base_oid); err != nil {
		log.Warningf("Error trying to delete application from db: %v", err)
	}
}

func (self *SNMPSubagent) initDB() error {
	db = nil
	db_file := self.config.DbFile
	if self.config.DbFile == "" {
		db_file = "data/applications.db"
	}

	var err error
	if db, err = initDB(db_file); err != nil {
		log.Error("Error initializing scripts DB: ", err)
		return err
	}

	if err = createTableApps(db); err != nil {
		log.Errorf("Error initializing applications table (db file: %v): %v", db_file, err)
		db.Close()
		db = nil
		return err
	}

	if err = createTableOids(db); err != nil {
		log.Error("Error initializing oids table (db file: %v): %v", db_file, err)
		db.Close()
		db = nil
		return err
	}

	return nil
}

func (self *SNMPSubagent) Run() {
	go self.serve()
}

type RegisterForm struct {
	Name               string        `json:"name" binding:"required"`
	DiscoverUrl        string        `json:"discover_url"`
	BaseOID            string        `json:"base_oid" binding:"required"`
	RediscoverInterval uint32        `json:"rediscover_interval"`
	Oids               []OIDResource `form:"oids" json:"oids",flow`
}

type UnregisterForm struct {
	BaseOID string `form:"base_oid" json:"base_oid" binding:"required"`
}

func (self *SNMPSubagent) serve() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(GinLogger())
	r.Use(gin.Recovery())
	r.PUT("/applications/", self.registrationHandler)

	r.DELETE("/applications/", self.unregistrationHandler)

	r.Run(self.config.EndpointAddress)
}

func (self *SNMPSubagent) registrationHandler(c *gin.Context) {
	if c.ContentType() != "application/json" {
		log.Warningf("Invalid content type '%v'", c.ContentType())
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Content-Type"})
		return
	}
	var app_info RegisterForm
	err := c.BindJSON(&app_info)
	if err != nil {
		log.Warningf("Error parsing registration: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid data"})
		return
	}

	log.Infof("New registration for App: '%v', Base OID: '%v', Discover Url: '%v'", app_info.Name, app_info.BaseOID, app_info.DiscoverUrl)
	base_oid, err := ParseOID(app_info.BaseOID)
	if err != nil {
		log.Warningf("Error registering app '%v': %v", app_info.Name, err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid oid"})
		return
	}

	var oids map[string]*OIDData
	if len(app_info.Oids) > 0 {
		oids = make(map[string]*OIDData)
		for _, oid_data_elem := range app_info.Oids {
			oid_data, err := NewOIDData(oid_data_elem.Oid, oid_data_elem.Type, oid_data_elem.Url, oid_data_elem.Jsonpath)
			if err != nil {
				log.Warningf("Error parsing oid resource %v: %v", oid_data_elem.Oid, err)
				continue
			} else {
				oids[oid_data.Oid.String()] = oid_data
			}
		}
	} else {
		oids = nil
	}

	err = self.Register(app_info.Name, app_info.DiscoverUrl, base_oid, app_info.RediscoverInterval, oids)
	if err != nil {
		log.Warningf("Couldn't register App '%v': %v", app_info.Name, err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Couldn't register app " + app_info.Name})
	} else {
		c.String(http.StatusAccepted, "")
	}
}

func (self *SNMPSubagent) unregistrationHandler(c *gin.Context) {
	if c.ContentType() != "application/json" {
		log.Warningf("Invalid content type '%v'", c.ContentType())
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Content-Type"})
		return
	}
	var app_info UnregisterForm
	err := c.BindJSON(&app_info)
	if err != nil {
		log.Warningf("Error parsing unregistration")
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid data"})

		return
	}
	log.Infof("Unregistration for app with OID '%v'", app_info.BaseOID)
	base_oid, err := ParseOID(app_info.BaseOID)
	if err != nil {
		log.Warningf("Error unregistering OID '%v': %v", app_info.BaseOID, err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid oid"})
	} else {
		self.Unregister(base_oid)
		c.String(http.StatusAccepted, "")
	}
}
