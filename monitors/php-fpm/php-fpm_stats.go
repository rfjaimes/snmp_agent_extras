package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/posteo/go-agentx/pdu"
)

type StatsData struct {
	FPM   map[string]map[string]DataOID
	Pool  map[string]map[string]DataOID
	Procs map[string]map[string]DataOID
}

type DataOID struct {
	Index int
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
	"master_index":   DataOID{1, pdu.VariableTypeInteger, "0"},
	"master_version": DataOID{2, pdu.VariableTypeOctetString, ""},
}

var StatsPool = map[string]DataOID{
	"index":                DataOID{1, pdu.VariableTypeInteger, "0"},
	"pool":                 DataOID{2, pdu.VariableTypeOctetString, ""},
	"socket_type":          DataOID{3, pdu.VariableTypeInteger, "0"},
	"main_addr":            DataOID{4, pdu.VariableTypeOctetString, ""},
	"main_por":             DataOID{5, pdu.VariableTypeInteger, "0"},
	"main_socke":           DataOID{6, pdu.VariableTypeOctetString, ""},
	"process_manager":      DataOID{7, pdu.VariableTypeInteger, "0"},
	"start_time":           DataOID{8, pdu.VariableTypeOctetString, ""},
	"start_since":          DataOID{9, pdu.VariableTypeTimeTicks, "0"},
	"accepted_conn":        DataOID{10, pdu.VariableTypeCounter64, "0"},
	"listen_queue":         DataOID{11, pdu.VariableTypeInteger, "0"},
	"max_listen_queue":     DataOID{12, pdu.VariableTypeCounter64, "0"},
	"listen_queue_len":     DataOID{13, pdu.VariableTypeInteger, "0"},
	"idle_processes":       DataOID{14, pdu.VariableTypeInteger, "0"},
	"active_processes":     DataOID{15, pdu.VariableTypeInteger, "0"},
	"total_processes":      DataOID{16, pdu.VariableTypeInteger, "0"},
	"max_active_processes": DataOID{17, pdu.VariableTypeCounter64, "0"},
	"max_children_reached": DataOID{18, pdu.VariableTypeCounter64, "0"},
	//"slow requests":        DataOID{19, pdu.VariableTypeInteger, "0"},
}

var StatsProcs = map[string]DataOID{
	"pid":                 DataOID{1, pdu.VariableTypeInteger, "0"},
	"state":               DataOID{2, pdu.VariableTypeInteger, "0"},
	"start_time":          DataOID{3, pdu.VariableTypeOctetString, ""},
	"start_since":         DataOID{4, pdu.VariableTypeTimeTicks, "0"},
	"requests":            DataOID{5, pdu.VariableTypeCounter64, "0"},
	"request_duration":    DataOID{6, pdu.VariableTypeCounter32, "0"},
	"request_method":      DataOID{7, pdu.VariableTypeInteger, "0"},
	"request_URI":         DataOID{8, pdu.VariableTypeOctetString, ""},
	"content_length":      DataOID{9, pdu.VariableTypeCounter32, "0"},
	"user":                DataOID{10, pdu.VariableTypeOctetString, ""},
	"script":              DataOID{11, pdu.VariableTypeOctetString, ""},
	"last_request_cpu":    DataOID{12, pdu.VariableTypeInteger, "0"},
	"last_request_memory": DataOID{13, pdu.VariableTypeCounter32, "0"},
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

	sm.data_enums = map[string]DataEnum{
		"2.1.3": DataEnum{"ipv4": 1, "ipv6": 2, "ipv46": 3, "unixSocket": 4},
		"2.1.7": DataEnum{"static": 1, "dynamic": 2, "onDemand": 3},
		"3.1.2": DataEnum{"idle": 1, "running": 2},
	}

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

func (self *StatsManager) ParseValues(index string, value string) (v int64, err error) {
	var data int64 = 255

	if typeEnum, ok := self.data_enums[index]; ok {
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

func (self *StatsManager) Load(serverCount int, data string) error {
	//new_stats := make(map[string]StatsData)

	svrCntIdx := strconv.Itoa(serverCount)

	blocks := strings.Split(data, "************************\n")

	stats := StatsData{}

	stats.Pool = make(map[string]map[string]DataOID)
	stats.Pool[svrCntIdx] = *CloneMapStats(StatsPool)

	stats.Procs = make(map[string]map[string]DataOID)
	stats.Procs[svrCntIdx] = *CloneMapStats(StatsProcs)

	var k, v string

	var countPools, countProcs int
	countPools = 0
	countProcs = 0

	for block_idx, block := range blocks {
		lines := strings.Split(block, "\n")

		if block_idx == 0 {
			countPools++
		} else {
			countProcs++
		}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, ":") {
				values := strings.Split(line, ":")
				k = strings.Replace(strings.TrimSpace(values[0]), " ", "_", -1)
				v = strings.TrimSpace(values[1])

				if block_idx == 0 {
					if dataOid, ok := stats.Pool[svrCntIdx][k]; ok {
						dataOid.Value = v
						stats.Pool[svrCntIdx][k] = dataOid

						idxStr := fmt.Sprintf("%d.%d.%d", serverCount, countProcs, dataOid.Index)
						log.Info("report oid %s", idxStr)
					} else {
						log.Warning("key not found in Pool: %s", k)
					}

				} else {
					if dataOid, ok := stats.Procs[svrCntIdx][k]; ok {
						dataOid.Value = v
						stats.Procs[svrCntIdx][k] = dataOid

						idxStr := fmt.Sprintf("%d.%d.%d", serverCount, countProcs, dataOid.Index)
						log.Info("report oid %s", idxStr)
					} else {
						log.Warning("key not found in Procs: %s", k)
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
	//statusCmd := "curl http://localhost/status?full"

	//out, err := exec.Command("bash", "-c", statusCmd).Output()

	out := `pool:                 www
process manager:      dynamic
start time:           23/Apr/2016:23:12:01 -0400
start since:          412
accepted conn:        9
listen queue:         0
max listen queue:     0
listen queue len:     128
idle processes:       4
active processes:     1
total processes:      5
max active processes: 1
max children reached: 0
slow requests:        0

************************
pid:                  20320
state:                Idle
start time:           23/Apr/2016:23:12:01 -0400
start since:          412
requests:             1
request duration:     273
request method:       -
request URI:          -
content length:       0
user:                 -
script:               -
last request cpu:     0.00
last request memory:  262144

************************
pid:                  20321
state:                Idle
start time:           23/Apr/2016:23:12:01 -0400
start since:          412
requests:             2
request duration:     167
request method:       GET
request URI:          /status
content length:       0
user:                 -
script:               /etc/nginx/html/status
last request cpu:     0.00
last request memory:  262144

************************
pid:                  20322
state:                Idle
start time:           23/Apr/2016:23:12:01 -0400
start since:          412
requests:             2
request duration:     382
request method:       GET
request URI:          /status?full
content length:       0
user:                 -
script:               /etc/nginx/html/status
last request cpu:     0.00
last request memory:  262144

************************
pid:                  20323
state:                Idle
start time:           23/Apr/2016:23:12:01 -0400
start since:          412
requests:             2
request duration:     70
request method:       -
request URI:          -
content length:       0
user:                 -
script:               -
last request cpu:     0.00
last request memory:  262144

************************
pid:                  20324
state:                Running
start time:           23/Apr/2016:23:12:01 -0400
start since:          412
requests:             2
request duration:     177
request method:       GET
request URI:          /status?full
content length:       0
user:                 -
script:               /etc/nginx/html/status
last request cpu:     0.00
last request memory:  0`

	if true {
		if err := self.Load(1, string(out)); err != nil {
			log.Warning("Couldn't load stats:", err)
		} else {
			log.Debug("Stats updated")
		}
	} else {
		//log.Errorf("Command returned with error: %v", err)
	}
}
