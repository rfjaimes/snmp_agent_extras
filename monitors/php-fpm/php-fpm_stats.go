package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/posteo/go-agentx/pdu"
)

type StatsData struct {
	ServType         int
	ServName         string
	ServAddr         string
	ServReqs         int
	ServComms        int
	ServState        int
	ServPartnerRole  int
	ServPartnerState int
}

type DataOID struct {
	Index int
	Type  pdu.VariableType
	Value string
}

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

func CloneMapStas(mapStats map[string]DataOID) *map[string]DataOID {
	if mapStats == nil {
		return nil
	} else {
		cloneMapStats := map[string]DataOID{}
		for k, v := range mapStats {
			cloneMapStats[k] = v.CloneDataOID()
		}
		return cloneMapStats
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

type DataEnum map[string]int

type StatsManager struct {
	last_updated time.Time
	stats        map[string]StatsData
	mutex        *sync.Mutex
	data_enums   map[int]DataEnum
}

func NewStatsManager() *StatsManager {
	sm := &StatsManager{}
	sm.last_updated = time.Unix(1, 1)
	sm.stats = make(map[string]StatsData)
	sm.mutex = &sync.Mutex{}

	sm.data_enums = map[int]DataEnum{
		1: DataEnum{"main": 1, "backup": 2, "dns": 3},
		5: DataEnum{"ok": 1, "interrupted": 2, "none": 3},
		6: DataEnum{"normal": 1, "recover": 2, "recover-done": 3, "partner-down": 4, "send-update": 5, "probe": 6},
		7: DataEnum{"main": 1, "backup": 2, "standalone": 3, "ha-main": 4, "ha-backup": 5},
		8: DataEnum{"normal": 1, "partner-down": 4, "communication-interrupted": 7, "paused": 8},
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

func (self *StatsManager) ParseValues(index int, value string) int64 {
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

	return data
}

func (self *StatsManager) Load(data string) error {
	blocks := strings.Split(data, "************************\n")

	new_stats := make(map[string]StatsData)

	for block_idx, block := range blocks {
		lines := strings.Split(block, "\n")
		for line_idx, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, ":") {
				stats := StatsData{}

				values := strings.Split(line, ":")

				stats_r := reflect.ValueOf(&stats).Elem()

				for i := 1; i <= 8; i++ {

					if stats_r.Field(i-1).Type() != reflect.TypeOf("") {
						valueEnum := self.ParseValues(i, strings.ToLower(values[i-1]))
						stats_r.Field(i - 1).SetInt(valueEnum)
					} else {
						stats_r.Field(i - 1).SetString(values[i-1])
					}

				}

				idxStr := fmt.Sprintf("%d.%d", block_idx, line_idx)
				new_stats[idxStr] = stats
			}

		}
	}

	self.mutex.Lock()
	self.stats = new_stats
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
		if err := self.Load(string(out)); err != nil {
			log.Warning("Couldn't load stats:", err)
		} else {
			log.Debug("Stats updated")
		}
	} else {
		//log.Errorf("Command returned with error: %v", err)
	}
}
