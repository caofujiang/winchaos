package category

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
)

func CpuRun(ctx context.Context, cpuCount int, cpuPercent int, uid string) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pid := os.Getpid()
	pidStr := strconv.FormatInt(int64(pid), 10)
	val := &Cpuparams{
		CpuCount:   cpuCount,
		CpuPercent: cpuPercent,
		PID:        pidStr,
		UID:        uid,
	}

	result, _ := json.Marshal(val)
	logrus.Info("cpu-main-param", uid, string(result))
	t1 := time.Now().Format(time.RFC3339Nano)
	commandModel := &data.ExperimentModel{
		Uid:        uid,
		Command:    "create cpu fullload",
		CmdType:    "cpu",
		Status:     Created,
		Flag:       string(result),
		Error:      "",
		CreateTime: t1,
		UpdateTime: t1,
	}
	checkError(GetDS().InsertExperimentModel(commandModel))
	param := &Cpuparams{
		CpuCount:   cpuCount,
		CpuPercent: cpuPercent,
	}
	tools.Go(context.Background(), func(ctx context.Context) func(ctx context.Context, closing <-chan struct{}) {
		return func(ctx context.Context, closing <-chan struct{}) {
			if err := CpuStart(ctx, param.CpuCount, param.CpuPercent, uid); err != nil {
				logrus.Errorf("start cpu failed", err.Error())
				return
			}
		}
	}(ctx))
	return
}

type Cpuparams struct {
	Cmt        ChaosbladeCPUType
	CpuCount   int    `json:"cpu-count"`
	CpuPercent int    `json:"cpu-percent"`
	Timeout    int    `json:"timeout"`
	PID        string `json:"pid"`
	UID        string `json:"uid"`
}

// CpuStart burn cpu
func CpuStart(ctx context.Context, cpuCount, cpuPercent int, uid string) error {
	ctx = context.WithValue(ctx, "cpuCount", cpuCount)
	runtime.GOMAXPROCS(cpuCount)
	logrus.Debugf("cpu counts: %d", cpuCount)
	slopePercent := float64(cpuPercent)
	climbTime := 0
	// make CPU slowly climb to some level, to simulate slow resource competition
	// which system faults cannot be quickly noticed by monitoring system.
	slope(ctx, cpuPercent, climbTime, &slopePercent)
	quota := make(chan int64, cpuCount)
	for i := 0; i < cpuCount; i++ {
		go burn(ctx, quota, slopePercent, uid)
	}

	for {
		q := getQuota(ctx, slopePercent)
		for i := 0; i < cpuCount; i++ {
			quota <- q
		}
	}
}

const period = int64(1000000000)

func slope(ctx context.Context, cpuPercent int, climbTime int, slopePercent *float64) {
	if climbTime != 0 {
		var ticker = time.NewTicker(time.Second)
		*slopePercent = getUsed(ctx)
		var startPercent = float64(cpuPercent) - *slopePercent
		go func() {
			for range ticker.C {
				if *slopePercent < float64(cpuPercent) {
					*slopePercent += startPercent / float64(climbTime)
				} else if *slopePercent > float64(cpuPercent) {
					*slopePercent -= startPercent / float64(climbTime)
				}
			}
		}()
	}
}

func getQuota(ctx context.Context, slopePercent float64) int64 {
	used := getUsed(ctx)
	dx := (slopePercent - used) / 100
	busy := int64(dx * float64(period))
	return busy
}

func burn(ctx context.Context, quota <-chan int64, slopePercent float64, uid string) {
	q := getQuota(ctx, slopePercent)
	ds := period - q
	if ds < 0 {
		ds = 0
	}
	s, _ := time.ParseDuration(strconv.FormatInt(ds, 10) + "ns")
	for {
		startTime := time.Now().UnixNano()
		t2 := time.Now().Second()

		select {
		case offset := <-quota:
			q = q + offset
			if q < 0 {
				q = 0
			}
			ds := period - q
			if ds < 0 {
				ds = 0
			}
		case <-ctx.Done():
			checkError(GetDS().UpdateExperimentModelByUid(uid, Success, ""))
			logrus.Info("cpu-burn-done", uid)
			os.Exit(0)
		default:
			for time.Now().UnixNano()-startTime < q {
			}
			runtime.Gosched()
			time.Sleep(s)
			if (time.Now().Second() - t2) > 5 {
				logrus.Info("cpu-default", time.Now().Second()-t2)
				return
			}
		}
	}
}

func getUsed(ctx context.Context) float64 {
	totalCpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		logrus.Errorf("get cpu usage fail, %s", err.Error())
	}
	return totalCpuPercent[0]
}

// checkError for db operation
func checkError(err error) {
	if err != nil {
		log.Println(context.Background(), err.Error())
	}
}

// sqlite
var ds data.SourceI

// GetDS returns dataSource
func GetDS() data.SourceI {
	if ds == nil {
		ds = data.GetSource()
	}
	return ds
}

const (
	Created   = "Created"
	Success   = "Success"
	Running   = "Running"
	Error     = "Error"
	Destroyed = "Destroyed"
	Revoked   = "Revoked"
)
