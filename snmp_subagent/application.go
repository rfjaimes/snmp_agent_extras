package snmp_subagent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"gitlab.intraway.com/golang/snmp-handler"

	agentx "github.com/posteo/go-agentx"
	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
)

type Application struct {
	name              string
	discover_url      string
	base_oid          value.OID
	discover_interval uint32
	client            *agentx.Client
	session           *agentx.Session
	handler           *snmp_handler.SNMPHandler
	oids              map[string]*OIDData
	get_timeout       uint32
	discover_timeout  uint32
	m                 *sync.Mutex
	rediscover_chan   chan bool
}

type OIDResource struct {
	Oid      string `json:"oid"`
	Type     string `json:"type"`
	Url      string `json:"url"`
	Jsonpath string `json:"jsonpath"`
}

type EmbeddedDiscovery struct {
	Oids []OIDResource `json:"oids",flow`
}

type DiscoveryGet struct {
	Status   string            `json:"status"`
	ErrCode  string            `json:"errcode"`
	ErrMsg   string            `json:"errmessage"`
	Embedded EmbeddedDiscovery `json:"_embedded",flow`
}

func NewApplication(name string,
	discover_url string,
	base_oid value.OID,
	discover_interval uint32,
	get_timeout uint32,
	discover_timeout uint32,
	oids map[string]*OIDData) *Application {
	app := &Application{
		name:              name,
		discover_url:      discover_url,
		base_oid:          base_oid,
		discover_interval: discover_interval,
		get_timeout:       get_timeout,
		discover_timeout:  discover_timeout,
		m:                 &sync.Mutex{},
		rediscover_chan:   make(chan bool),
	}

	app.handler = snmp_handler.NewSNMPHandler(base_oid)
	app.oids = make(map[string]*OIDData)

	if oids != nil {
		for _, oid_data := range oids {
			err := app.handler.Register(oid_data.Oid, oid_data.Type, app)
			if err != nil {
				log.Warningf("Error registering oid: %v", err)
			} else {
				app.oids[oid_data.Oid.String()] = oid_data
				log.Infof("Oid %v successfully registered for %v", oid_data.Oid, name)
			}
		}
	}

	return app
}

func (self *Application) RemoveAllOids() {
	self.oids = make(map[string]*OIDData)
}

func ParseDiscoveryResponse(discovery_body []byte) (DiscoveryGet, error) {
	var resource DiscoveryGet
	err := json.Unmarshal(discovery_body, &resource)
	if err != nil {
		return resource, err
	}

	return resource, nil
}

// Getter interface (snmp_handler)
func (self Application) Get(oid value.OID) interface{} {
	client := http.Client{
		Timeout: time.Duration(self.get_timeout) * time.Millisecond,
	}

	oid_data, ok := self.oids[oid.String()]
	if !ok {
		log.Errorf("OID %v not registered. This shouldn't happen", oid)
		return nil
	}

	endpoint := oid_data.Url
	log.Debugf("Trying to retrieve OID %v with endpoint %v", oid, endpoint)

	resp, err := client.Get(endpoint)
	if err != nil {
		log.Warningf("Error while trying to retrieve oid %v: %v", oid, err)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Warningf("Server responded with status '%v'", resp.Status)
		return nil
	}

	content_type := filterFlags(resp.Header.Get("Content-Type"))
	if content_type != "application/json" {
		log.Warningf("Invalid content type '%v'", content_type)
		return nil
	}

	oid_body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Warningf("Error while trying to retrieve oid %v: %v", oid, err)
		return nil
	}

	value, err := processOidResponse(oid_body, oid_data)
	if err != nil {
		log.Warningf("Error processing oid response of oid %v: %v", oid, err)
		return nil
	}

	return value
	return nil
}

func processOidResponse(oid_body []byte, oid_data *OIDData) (interface{}, error) {
	var json_data interface{}
	err := json.Unmarshal(oid_body, &json_data)
	if err != nil {
		return nil, err
	}

	val, err := oid_data.Applicator.Apply(json_data)
	if err != nil {
		return nil, err
	}

	// parse and convert value
	switch oid_data.Type {
	case pdu.VariableTypeOctetString:
		switch t := val.(type) {
		case string:
			return val.(string), nil
		default:
			return nil, fmt.Errorf("Value '%v' is not a string (%v)", val, t)
		}
	case pdu.VariableTypeInteger:
		switch t := val.(type) {
		case string:
			i, err := strconv.ParseInt(val.(string), 10, 32)
			if err != nil {
				return nil, err
			} else {
				return int32(i), nil
			}
		case float64:
			if val.(float64) <= math.MaxInt32 && val.(float64) >= math.MinInt32 {
				return int32(val.(float64)), nil
			} else {
				return nil, fmt.Errorf("%v is not within int32 boundaries", val)
			}
		default:
			return nil, fmt.Errorf("Wrong type for value '%v' (%v)", val, t)
		}
	case pdu.VariableTypeObjectIdentifier:
		switch t := val.(type) {
		case string:
			_, err := ParseOID(val.(string))
			if err != nil {
				return nil, err
			} else {
				return val.(string), nil
			}
		default:
			return nil, fmt.Errorf("Value '%v' is not a string (%v)", val, t)
		}
	case pdu.VariableTypeIPAddress:
		switch t := val.(type) {
		case string:
			ip := net.ParseIP(val.(string))
			if ip == nil {
				return nil, fmt.Errorf("%v is not a valid IP address", val)
			} else {
				return ip.To4(), nil
			}
		default:
			return nil, fmt.Errorf("Value '%v' is not a string (%v)", val, t)
		}
	case pdu.VariableTypeCounter32, pdu.VariableTypeGauge32:
		switch t := val.(type) {
		case string:
			i, err := strconv.ParseUint(val.(string), 10, 32)
			if err != nil {
				return nil, err
			} else {
				return uint32(i), nil
			}
		case float64:
			if val.(float64) <= math.MaxUint32 && val.(float64) >= 0 {
				return uint32(val.(float64)), nil
			} else {
				return nil, fmt.Errorf("%v is not within uint32 boundaries", val)
			}
		default:
			return nil, fmt.Errorf("Wrong type for value '%v' (%v)", val, t)
		}
	case pdu.VariableTypeTimeTicks:
		switch t := val.(type) {
		case string:
			// RFC 2578: https://tools.ietf.org/html/rfc2578#section-7.1.8
			// The TimeTicks type represents a non-negative integer which represents the time, modulo 2^32 (4294967296 decimal)
			i, err := strconv.ParseUint(val.(string), 10, 32)
			if err != nil {
				return nil, err
			} else {
				return time.Duration(int64(i) * int64(time.Second)), nil
			}
		case float64:
			if val.(float64) <= math.MaxInt64 && val.(float64) >= 0 {
				return time.Duration(int64(val.(float64)) * int64(time.Second)), nil
			} else {
				return nil, fmt.Errorf("%v is not within uint64 boundaries", val)
			}
		default:
			return nil, fmt.Errorf("Wrong type for value '%v' (%v)", val, t)
		}
	case pdu.VariableTypeOpaque:
		switch t := val.(type) {
		case string:
			// Ponele...
			return []byte(val.(string)), nil
		default:
			return nil, fmt.Errorf("Value '%v' is not a string (%v)", val, t)
		}
	case pdu.VariableTypeCounter64:
		switch t := val.(type) {
		case string:
			i, err := strconv.ParseUint(val.(string), 10, 64)
			if err != nil {
				return nil, err
			} else {
				return uint64(i), nil
			}
		case float64:
			if val.(float64) <= math.MaxUint64 && val.(float64) >= 0 {
				return uint64(val.(float64)), nil
			} else {
				return nil, fmt.Errorf("%v is not within uint64 boundaries", val)
			}
		default:
			return nil, fmt.Errorf("Wrong type for value '%v' (%v)", val, t)
		}
	}

	return val, nil
}

func (self *Application) discover() {
	self.m.Lock()
	defer self.m.Unlock()

	endpoint := self.discover_url
	log.Info("Discovery of", self.name, "with endpoint ", endpoint)

	client := http.Client{
		Timeout: time.Duration(self.discover_timeout) * time.Millisecond,
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		log.Warningf("Error while trying to discover %v: %v", self.name, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Warningf("Server responded with status '%v'", resp.Status)
		return
	}

	discovery, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Warningf("Error while trying to discover %v: %v", self.name, err)
		return
	}

	discovery_resource, err := ParseDiscoveryResponse(discovery)
	if err != nil {
		log.Warningf("Error processing discover response of %v: %v", self.name, err)
		return
	}

	self.handler.UnregisterAll()
	self.RemoveAllOids()

	oids := discovery_resource.Embedded.Oids
	if len(oids) == 0 {
		log.Warningf("Error empty oids in discovery of %v", self.name)
	}

	for _, oid_resource := range oids {
		oid_data, err := NewOIDData(oid_resource.Oid, oid_resource.Type, oid_resource.Url, oid_resource.Jsonpath)
		if err != nil {
			log.Warningf("Error parsing oid resource %v: %v", oid_resource.Oid, err)
			continue
		}

		err = self.handler.Register(oid_data.Oid, oid_data.Type, self)
		if err != nil {
			log.Warningf("Error registering oid: %v", err)
		} else {
			self.oids[oid_data.Oid.String()] = oid_data
			log.Infof("Oid %v successfully registered for %v", oid_data.Oid, self.name)
		}
	}
}

func (self *Application) stopRediscover() {
	self.m.Lock()
	defer self.m.Unlock()

	// Non-blocking send
	select {
	case self.rediscover_chan <- true:
	default:
	}
}

func (self *Application) rediscover() {
	ticker := time.NewTicker(time.Second * time.Duration(self.discover_interval))
	quit := false
	for {
		if quit {
			break
		}

		select {
		case <-ticker.C:
			log.Infof("Rediscovering application %v", self.name)
			self.discover()
		case <-self.rediscover_chan:
			quit = true
		}
	}
}
