package stats

import (
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("iostat_monitor")

type StatsData struct {
	Device  string
	Rrqms   float32
	Wrqms   float32
	Rs      float32
	Ws      float32
	Rkbs    float32
	Wkbs    float32
	Avgrqsz float32
	Avgqusz float32
	Await   float32
	Rawait  float32
	Wawait  float32
	Svctm   float32
	Util    float32
}

type StatsManager struct {
	last_updated time.Time
	stats        map[string]StatsData
	mutex        *sync.Mutex
}

func NewStatsManager() *StatsManager {
	sm := &StatsManager{}
	sm.last_updated = time.Unix(1, 1)
	sm.stats = make(map[string]StatsData)
	sm.mutex = &sync.Mutex{}
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

var IOSTAT_LINE_RE = regexp.MustCompile(`^\s*([^ ]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s+([0-9\.]+)\s*$`)

func (self *StatsManager) Load(data string) error {
	lines := strings.Split(data, "\n")
	new_stats := make(map[string]StatsData)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		m := IOSTAT_LINE_RE.FindStringSubmatch(line)
		if len(m) != 15 {
			return fmt.Errorf("Wrong data format: %v", line)
		}

		stats := StatsData{}
		stats.Device = m[1]

		stats_r := reflect.ValueOf(&stats).Elem()

		for i := 2; i <= 14; i++ {
			if f, err := strconv.ParseFloat(m[i], 32); err != nil {
				return err
			} else {
				stats_r.Field(i - 1).SetFloat(f)
			}
		}

		new_stats[m[1]] = stats
	}

	self.mutex.Lock()
	self.stats = new_stats
	self.last_updated = time.Now()
	self.mutex.Unlock()
	return nil
}

var IOSTAT_RE = regexp.MustCompile(`(?s)^.*util\s*(.*)\s+$`)

func (self *StatsManager) Run(interval_watch uint, interval_exec uint) {
	log.Infof("Running stats loader. Watch stats during %d seconds. Repeat every %d seconds", interval_watch, interval_exec)
	ticker := time.NewTicker(time.Second * time.Duration(interval_exec))
	go func() {
		for _ = range ticker.C {
			out, err := exec.Command("iostat", "-xkd", strconv.Itoa(int(interval_watch)), "2").Output()
			if err == nil {
				m := IOSTAT_RE.FindStringSubmatch(string(out))
				if len(m) == 2 {
					if err := self.Load(strings.Replace(m[1], ",", ".", -1)); err != nil {
						log.Warning("Couldn't load stats:", err)
					} else {
						log.Debug("Stats updated")
					}
				} else {
					log.Error("Invalid output from command")
				}
			} else {
				log.Errorf("Command returned with error: %v", err)
			}
		}
	}()
}
