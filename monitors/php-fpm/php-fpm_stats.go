package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/posteo/go-agentx/pdu"
)

type StatsData struct {
	FPM   map[string]map[string]DataOID
	Pools map[string]map[string]DataOID
	Procs map[string]map[string]DataOID
}

type DataOID struct {
	Index string
	Type  pdu.VariableType
	Value string
}

type DataEnum map[string]int

func (self *DataOID) CloneDataOID() *DataOID {
	if self == nil {
		return nil
	} else {
		cloneDataOid := &DataOID{
			self.Index,
			self.Type,
			self.Value,
		}
		return cloneDataOid
	}
}

var dataEnums = map[string]DataEnum{
	"2.1.3": DataEnum{"ipv4": 1, "ipv6": 2, "ipv46": 3, "unixsocket": 4},
	"2.1.7": DataEnum{"static": 1, "dynamic": 2, "ondemand": 3},
	"3.1.2": DataEnum{"idle": 1, "running": 2},
}

func (self *DataOID) ParseValue(index string, value string) (v int64, err error) {
	var data int64 = 255

	if typeEnum, ok := dataEnums[index]; ok {
		if valueEnum, ok := typeEnum[value]; ok {
			data = int64(valueEnum)
		}
	} else {
		dEnum, err := strconv.Atoi(value)
		if err == nil {
			data = int64(dEnum)
		}
	}

	return data, err
}

func (self *DataOID) getOIDValue() interface{} {
	var oidValue interface{}
	switch self.Type {
	case pdu.VariableTypeInteger:
		if valueOid, err := self.ParseValue(self.Index, strings.ToLower(self.Value)); err == nil {
			oidValue = int32(valueOid)
		}
	case pdu.VariableTypeOctetString:
		oidValue = self.Value
	case pdu.VariableTypeObjectIdentifier:
		oidValue = self.Value
	case pdu.VariableTypeIPAddress:
		oidValue = net.IP{10, 10, 10, 10}
	case pdu.VariableTypeCounter32:
		if valueOid, err := strconv.ParseInt(self.Value, 10, 64); err == nil {
			oidValue = uint32(valueOid)
		}
	case pdu.VariableTypeGauge32:
		if valueOid, err := strconv.ParseInt(self.Value, 10, 64); err == nil {
			oidValue = uint32(valueOid)
		}
	case pdu.VariableTypeTimeTicks:
		if valueOid, err := strconv.ParseInt(self.Value, 10, 64); err == nil {
			oidValue = time.Duration(valueOid) * time.Second
		}
	case pdu.VariableTypeCounter64:
		if valueOid, err := strconv.ParseInt(self.Value, 10, 64); err == nil {
			oidValue = uint64(valueOid)
		}
	}
	return oidValue
}

func CloneMapStats(mapStats map[string]DataOID) *map[string]DataOID {
	if mapStats == nil {
		return nil
	} else {
		cloneMapStats := map[string]DataOID{}
		for k, v := range mapStats {
			cloneMapStats[k] = *v.CloneDataOID()
		}
		return &cloneMapStats
	}
}

var StatsFPM = map[string]DataOID{
	"master_index":   DataOID{"1.1.1", pdu.VariableTypeInteger, "0"},
	"master_version": DataOID{"1.1.2", pdu.VariableTypeOctetString, ""},
}

var StatsPool = map[string]DataOID{
	"index":                DataOID{"2.1.1", pdu.VariableTypeInteger, "0"},
	"pool":                 DataOID{"2.1.2", pdu.VariableTypeOctetString, ""},
	"socket_type":          DataOID{"2.1.3", pdu.VariableTypeInteger, "0"},
	"main_addr":            DataOID{"2.1.4", pdu.VariableTypeOctetString, ""},
	"main_por":             DataOID{"2.1.5", pdu.VariableTypeInteger, "0"},
	"main_socke":           DataOID{"2.1.6", pdu.VariableTypeOctetString, ""},
	"process_manager":      DataOID{"2.1.7", pdu.VariableTypeInteger, "0"},
	"start_time":           DataOID{"2.1.8", pdu.VariableTypeOctetString, ""},
	"start_since":          DataOID{"2.1.9", pdu.VariableTypeTimeTicks, "0"},
	"accepted_conn":        DataOID{"2.1.10", pdu.VariableTypeCounter64, "0"},
	"listen_queue":         DataOID{"2.1.11", pdu.VariableTypeInteger, "0"},
	"max_listen_queue":     DataOID{"2.1.12", pdu.VariableTypeCounter64, "0"},
	"listen_queue_len":     DataOID{"2.1.13", pdu.VariableTypeInteger, "0"},
	"idle_processes":       DataOID{"2.1.14", pdu.VariableTypeInteger, "0"},
	"active_processes":     DataOID{"2.1.15", pdu.VariableTypeInteger, "0"},
	"total_processes":      DataOID{"2.1.16", pdu.VariableTypeInteger, "0"},
	"max_active_processes": DataOID{"2.1.17", pdu.VariableTypeCounter64, "0"},
	"max_children_reached": DataOID{"2.1.18", pdu.VariableTypeCounter64, "0"},
	//"slow requests":        DataOID{19, pdu.VariableTypeInteger, "0"},
}

var StatsProcs = map[string]DataOID{
	"pid":                 DataOID{"3.1.1", pdu.VariableTypeInteger, "0"},
	"state":               DataOID{"3.1.2", pdu.VariableTypeInteger, "0"},
	"start_time":          DataOID{"3.1.3", pdu.VariableTypeOctetString, ""},
	"start_since":         DataOID{"3.1.4", pdu.VariableTypeTimeTicks, "0"},
	"requests":            DataOID{"3.1.5", pdu.VariableTypeCounter64, "0"},
	"request_duration":    DataOID{"3.1.6", pdu.VariableTypeCounter32, "0"},
	"request_method":      DataOID{"3.1.7", pdu.VariableTypeInteger, "0"},
	"request_URI":         DataOID{"3.1.8", pdu.VariableTypeOctetString, ""},
	"content_length":      DataOID{"3.1.9", pdu.VariableTypeCounter32, "0"},
	"user":                DataOID{"3.1.10", pdu.VariableTypeOctetString, ""},
	"script":              DataOID{"3.1.11", pdu.VariableTypeOctetString, ""},
	"last_request_cpu":    DataOID{"3.1.12", pdu.VariableTypeInteger, "0"},
	"last_request_memory": DataOID{"3.1.13", pdu.VariableTypeCounter32, "0"},
}

type StatsManager struct {
	last_updated time.Time
	stats        map[string]StatsData
	mutex        *sync.Mutex
	data_enums   map[string]DataEnum
	serversCount int
}

func NewStatsManager() *StatsManager {
	sm := &StatsManager{}
	sm.last_updated = time.Unix(1, 1)
	sm.stats = make(map[string]StatsData)
	sm.mutex = &sync.Mutex{}
	sm.serversCount = 0

	return sm
}

func (self *StatsManager) Stats() map[string]StatsData {
	self.mutex.Lock()
	stats := self.stats
	self.mutex.Unlock()
	return stats
}

func (self *StatsManager) LastUpdated() time.Time {
	self.mutex.Lock()
	lu := self.last_updated
	self.mutex.Unlock()
	return lu
}

func (self *StatsManager) Load(serverCount int, data string) error {

	svrCntIdx := strconv.Itoa(serverCount)

	blocks := strings.Split(data, "************************\n")

	stats := StatsData{}

	stats.FPM = make(map[string]map[string]DataOID)
	stats.Pools = make(map[string]map[string]DataOID)
	stats.Procs = make(map[string]map[string]DataOID)

	var k, v, strCountPools, strCountProcs string

	var countPools, countProcs int
	countPools = 0
	countProcs = 0

	stats.FPM[svrCntIdx] = *CloneMapStats(StatsFPM)

	if dataOid, ok := stats.FPM[svrCntIdx]["master_index"]; ok {
		dataOid.Value = svrCntIdx
		stats.FPM[svrCntIdx]["master_index"] = dataOid
	}

	if dataOid, ok := stats.FPM[svrCntIdx]["master_version"]; ok {
		dataOid.Value = "v0.0"
		stats.FPM[svrCntIdx]["master_version"] = dataOid
	}

	for block_idx, block := range blocks {
		lines := strings.Split(block, "\n")

		if block_idx == 0 {
			countPools++
			strCountPools = strconv.Itoa(countPools)
			stats.Pools[strCountPools] = *CloneMapStats(StatsPool)

		} else {
			countProcs++
			strCountProcs = strconv.Itoa(countProcs)
			stats.Procs[strCountProcs] = *CloneMapStats(StatsProcs)
		}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, ":") {
				values := strings.Split(line, ":")
				k = strings.Replace(strings.TrimSpace(values[0]), " ", "_", -1)
				v = strings.TrimSpace(values[1])

				if block_idx == 0 {
					if dataOid, ok := stats.Pools[strCountPools][k]; ok {
						dataOid.Value = v
						stats.Pools[strCountPools][k] = dataOid

						idxStr := fmt.Sprintf("%s.%d.%d", dataOid.Index, serverCount, countPools)
						log.Infof("read oid %s", idxStr)
					} else {
						log.Warningf("key not found in Pool: %s", k)
					}

				} else {
					if dataOid, ok := stats.Procs[strCountProcs][k]; ok {
						dataOid.Value = v
						stats.Procs[strCountProcs][k] = dataOid

						idxStr := fmt.Sprintf("%s.%d.%d", dataOid.Index, serverCount, countProcs)
						log.Infof("read oid %s", idxStr)
					} else {
						log.Warningf("key not found in Procs: %s", k)
					}

				}

			}

		}
	}

	self.mutex.Lock()
	self.stats[svrCntIdx] = stats
	self.last_updated = time.Now()
	self.mutex.Unlock()
	return nil
}

func (self *StatsManager) Run(interval_exec uint) {
	log.Infof("Running stats loader. Repeat every %d seconds", interval_exec)
	self.Execute()

	ticker := time.NewTicker(time.Second * time.Duration(interval_exec))
	go func() {
		for _ = range ticker.C {
			self.Execute()
		}
	}()
}

func (self *StatsManager) Execute() {
	statusCmd := "curl http://localhost/status?full"

	out, err := exec.Command("bash", "-c", statusCmd).Output()

	if true {
		if err := self.Load(1, string(out)); err != nil {
			log.Warning("Couldn't load stats:", err)
		} else {
			log.Debug("Stats updated")
		}
	} else {
		log.Errorf("Command returned with error: %v", err)
	}
}
