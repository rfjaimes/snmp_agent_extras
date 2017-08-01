package stats

import (
	"fmt"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("iostat_monitor")

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
	lines := strings.Split(data, "\n")
	new_stats := make(map[string]StatsData)
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ";") {
			stats := StatsData{}

			values := strings.Split(line, ";")

			stats_r := reflect.ValueOf(&stats).Elem()

			for i := 1; i <= 8; i++ {

				if stats_r.Field(i-1).Type() != reflect.TypeOf("") {
					valueEnum := self.ParseValues(i, strings.ToLower(values[i-1]))
					stats_r.Field(i - 1).SetInt(valueEnum)
				} else {
					stats_r.Field(i - 1).SetString(values[i-1])
				}

			}

			idxStr := fmt.Sprintf("%d", idx)
			new_stats[idxStr] = stats
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
	out, err := exec.Command("bash", "-c", `nrcmd -N admin -P changeme -r "dhcp getRelatedServers column-separator=\|" | \
    egrep '^MAIN|^BACKUP|^DNS' | \
    awk 'function ltrim(s) { sub(/^[ \t\r\n]+/, "", s); return s }
function rtrim(s) { sub(/[ \t\r\n]+$/, "", s); return s }
function trim(s) { return rtrim(ltrim(s)); }
BEGIN{FS="|"}{print trim($1)";"trim($2)";"trim($3)";"trim($4)";"trim($5)";"trim($6)";"trim($7)";"trim($8)}'`)

	//     out := `BACKUP;iw-cnr1-2.libertypr.com;10.229.198.25;0;OK;NORMAL;BACKUP;NORMAL
	// BACKUP;iw-cnr1-3.libertypr.com;10.229.198.26;0;INTERRUPTED;RECOVER;MAIN;COMMUNICATION-INTERRUPTED
	// DNS;dns0.libertypr.com;10.229.198.30;0;OK;SEND-UPDATE;STANDALONE;--
	// DNS;iway-backend-01;10.10.35.133;0;OK;SEND-UPDATE;HA-MAIN;--
	// DNS;iway-backend-02;10.10.35.134;0;NONE;PROBE;HA-BACKUP;--`

	if err == nil {
		if err := self.Load(out); err != nil {
			log.Warning("Couldn't load stats:", err)
		} else {
			log.Debug("Stats updated")
		}
	} else {
		log.Errorf("Command returned with error: %v", err)
	}
}
