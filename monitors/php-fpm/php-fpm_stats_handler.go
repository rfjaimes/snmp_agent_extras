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

	dataStats := self.stats_manager.Stats()

	keys := make([]string, len(dataStats))
	i := 0
	for k := range dataStats {
		keys[i] = k
		i++
	}

	sort.Strings(keys)

	for svr_idx, device := range keys {
		stats_data := dataStats[device]

		for fpm_idx, fpm := range stats_data.FPM {

			for _, dataOid := range fpm {
				oid := self.base_oid + "." + dataOid.Index + "." + strconv.Itoa(svr_idx+1) + "." + fpm_idx
				self.oids = append(self.oids, value.MustParseOID(oid))

				self.items[oid] = agentx.ListItem{dataOid.Type, dataOid.getOIDValue()}

				log.Infof("write oid %s : %s", oid, dataOid.Value)
			}
		}

		for pool_idx, pool := range stats_data.Pools {

			for _, dataOid := range pool {
				oid := self.base_oid + "." + dataOid.Index + "." + strconv.Itoa(svr_idx+1) + "." + pool_idx
				self.oids = append(self.oids, value.MustParseOID(oid))

				self.items[oid] = agentx.ListItem{dataOid.Type, dataOid.getOIDValue()}

				log.Infof("write oid %s : %s", oid, dataOid.Value)
			}
		}

		for proc_idx, proc := range stats_data.Procs {

			for _, dataOid := range proc {
				oid := self.base_oid + "." + dataOid.Index + "." + strconv.Itoa(svr_idx+1) + "." + proc_idx
				self.oids = append(self.oids, value.MustParseOID(oid))

				self.items[oid] = agentx.ListItem{dataOid.Type, dataOid.getOIDValue()}

				log.Infof("write oid %s : %s", oid, dataOid.Value)
			}
		}
	}

	sort.Sort(stats.OIDSorter(self.oids))
}
