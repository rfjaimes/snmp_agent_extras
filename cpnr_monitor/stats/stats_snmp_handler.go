package stats

import (
	"sort"
	"time"

	agentx "github.com/posteo/go-agentx"
	"github.com/posteo/go-agentx/pdu"
	"github.com/posteo/go-agentx/value"
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
		60,
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
		greater_than_from := OIDGreaterThan(oid, from)
		less_than_from := OIDLessThan(oid, from)
		less_than_to := OIDLessThan(oid, to)

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

	// for idx, device := range keys {
	// 	stats_data := stats[device]
	// 	// device index
	// 	oid := self.base_oid + ".1." + strconv.Itoa(idx+1)
	// 	self.oids = append(self.oids, value.MustParseOID(oid))
	// 	self.items[oid] = agentx.ListItem{pdu.VariableTypeInteger, int32(idx + 1)}

	// 	// device name
	// 	oid = self.base_oid + ".2." + strconv.Itoa(idx+1)
	// 	self.oids = append(self.oids, value.MustParseOID(oid))
	// 	self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, device}

	// // rrqm/s
	// oid = self.base_oid + ".3." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Rrqms)}

	// // wrqm/s
	// oid = self.base_oid + ".4." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Wrqms)}

	// // r/s
	// oid = self.base_oid + ".5." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Rs)}

	// // w/s
	// oid = self.base_oid + ".6." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Ws)}

	// // rkB/s
	// oid = self.base_oid + ".7." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Rkbs)}

	// // wkB/s
	// oid = self.base_oid + ".8." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Wkbs)}

	// // avgrq-sz
	// oid = self.base_oid + ".9." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Avgrqsz)}

	// // avgqu-sz
	// oid = self.base_oid + ".10." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Avgqusz)}

	// // await
	// oid = self.base_oid + ".11." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Await)}

	// // r_await
	// oid = self.base_oid + ".12." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Rawait)}

	// // w_await
	// oid = self.base_oid + ".13." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Wawait)}

	// // svctm
	// oid = self.base_oid + ".14." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Svctm)}

	// // %util
	// oid = self.base_oid + ".15." + strconv.Itoa(idx+1)
	// self.oids = append(self.oids, value.MustParseOID(oid))
	// self.items[oid] = agentx.ListItem{pdu.VariableTypeOctetString, fmt.Sprintf("%.2f", stats_data.Util)}
	//}
	sort.Sort(OIDSorter(self.oids))
}
