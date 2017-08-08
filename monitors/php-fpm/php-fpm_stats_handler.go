package main

import (
	"net"
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

		for pool_idx, pool := range stats_data.Pool {

			for _, dataOid := range pool {
				oid := self.base_oid + ".2.1." + strconv.Itoa(dataOid.Index) + "." + strconv.Itoa(svr_idx+1) + "." + pool_idx
				self.oids = append(self.oids, value.MustParseOID(oid))

				var oidValue interface{}

				switch dataOid.Type {
				case pdu.VariableTypeInteger:
					if valueOid, err := self.stats_manager.ParseValues("2.1."+strconv.Itoa(dataOid.Index), dataOid.Value); err == nil {
						oidValue = int32(valueOid)
					}
				case pdu.VariableTypeOctetString:
					oidValue = dataOid.Value
				case pdu.VariableTypeObjectIdentifier:
					oidValue = dataOid.Value
				case pdu.VariableTypeIPAddress:
					oidValue = net.IP{10, 10, 10, 10}
				case pdu.VariableTypeCounter32:
					if valueOid, err := strconv.ParseInt(dataOid.Value, 10, 64); err == nil {
						oidValue = uint32(valueOid)
					}
				case pdu.VariableTypeGauge32:
					if valueOid, err := strconv.ParseInt(dataOid.Value, 10, 64); err == nil {
						oidValue = uint32(valueOid)
					}
				case pdu.VariableTypeTimeTicks:
					if valueOid, err := strconv.ParseInt(dataOid.Value, 10, 64); err == nil {
						oidValue = int64(valueOid) * time.Second
					}
				case pdu.VariableTypeCounter64:
					if valueOid, err := strconv.ParseInt(dataOid.Value, 10, 64); err == nil {
						oidValue = uint64(valueOid)
					}
				}

				self.items[oid] = agentx.ListItem{dataOid.Type, oidValue}

				log.Info("report oid %s : %s", oid, dataOid.Value)
			}
		}

	}

	sort.Sort(stats.OIDSorter(self.oids))
}
