package main

import (
	"sort"
	"strconv"
	"time"

	agentx "github.com/posteo/go-agentx"
	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
	"github.com/rfjaimes/snmp_agent_extras/monitors/cpnr/stats"
)

type StatsSNMPHandler struct {
	stats_manager  *StatsManager
	base_oid       string
	oldest_allowed uint
	last_updated   time.Time
	oids           []value.OID
	items          map[string]agentx.ListItem
}

func NewStatsSNMPHandler(stats_manager *StatsManager, base_oid string) *StatsSNMPHandler {
	h := &StatsSNMPHandler{stats_manager,
		base_oid,
		90,
		time.Unix(1, 1),
		make([]value.OID, 0),
		make(map[string]agentx.ListItem)}
	return h
}

func (self *StatsSNMPHandler) Get(oid value.OID) (value.OID, pdu.VariableType, interface{}, error) {
	log.Debug("SNMP GET", oid)
	self.update()

	return self.doGet(oid)
}

func (self *StatsSNMPHandler) doGet(oid value.OID) (value.OID, pdu.VariableType, interface{}, error) {
	item, ok := self.items[oid.String()]
	if ok {
		return oid, item.Type, item.Value, nil
	} else {
		return nil, pdu.VariableTypeNoSuchObject, nil, nil
	}
}

func (self *StatsSNMPHandler) GetNext(from value.OID, include_from bool, to value.OID) (value.OID, pdu.VariableType, interface{}, error) {
	log.Debug("SNMP GETNEXT", from)
	self.update()
	if len(self.items) == 0 {
		return nil, pdu.VariableTypeNoSuchObject, nil, nil
	}

	for _, oid := range self.oids {
		greater_than_from := stats.OIDGreaterThan(oid, from)
		less_than_from := stats.OIDLessThan(oid, from)
		less_than_to := stats.OIDLessThan(oid, to)

		if greater_than_from && less_than_to {
			return self.doGet(oid)
		}

		// false with less and greater means equal
		if include_from && !less_than_from && !greater_than_from {
			return self.doGet(oid)
		}
	}

	return nil, pdu.VariableTypeNoSuchObject, nil, nil
}

func (self *StatsSNMPHandler) update() {
	if time.Since(self.last_updated).Seconds() < 3 {
		return
	}

	self.oids = make([]value.OID, 0)
	self.items = make(map[string]agentx.ListItem)
	self.last_updated = time.Now()

	if time.Since(self.stats_manager.LastUpdated()).Seconds() > float64(self.oldest_allowed) {
		log.Warning("Stats are not updated")
		return
	}

	stats := self.stats_manager.Stats()

	keys := make([]string, len(stats))
	i := 0
	for k := range stats {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	for idx, device := range keys {
		stats_data := stats[device]
		// Server index
		oid := self.base_oid + ".1." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(idx + 1)}

		// Server Type
		oid = self.base_oid + ".2." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(stats_data.ServType)}

		// Server Name
		oid = self.base_oid + ".3." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, stats_data.ServName}

		// Server Addr
		oid = self.base_oid + ".4." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, stats_data.ServAddr}

		// Server Reqs
		oid = self.base_oid + ".5." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(stats_data.ServReqs)}

		// Server Comms
		oid = self.base_oid + ".6." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(stats_data.ServComms)}

		// Server State
		oid = self.base_oid + ".7." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(stats_data.ServState)}

		// Server PartnerRole
		oid = self.base_oid + ".8." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(stats_data.ServPartnerRole)}

		// Server PartnerState
		oid = self.base_oid + ".9." + strconv.Itoa(idx+1)
		self.oids = append(self.oids, value.MustParseOID(oid))
		self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(stats_data.ServPartnerState)}

	}

	//sort.Sort(stats.OIDSorter(self.oids))
}
