package cmdexec

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/pkg/tools"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
)

type Cpuparam struct {
	Cmt        category.ChaosbladeCPUType
	CpuCount   int `json:"cpuCount"`
	CpuPercent int `json:"cpuPercent"`
	TimeOut    int `json:"timeOut"`
}

func CpuResolver(cpuParam *Cpuparam) (response *transport.Response) {
	if cpuParam == nil {
		logrus.Errorf("cpuParam is nil")
		return nil
	}
	switch cpuParam.Cmt {
	case category.ChaosbladeTypeCPUFullLoad:
		if cpuParam.CpuPercent != 0 {
			if cpuParam.CpuPercent > 100 || cpuParam.CpuPercent < 0 {
				logrus.Errorf("`%d`: cpu-percent is illegal, it must be a positive integer and not bigger than 100", cpuParam.CpuPercent)
				return nil
			}
		}
		if cpuParam.CpuCount != 0 {
			if cpuParam.CpuCount <= 0 || cpuParam.CpuCount > runtime.NumCPU() {
				cpuParam.CpuCount = runtime.NumCPU()
			}
		}
		var timeout time.Duration
		if cpuParam.TimeOut == 0 {
			// 默认超时
			timeout = 60 * time.Second
		} else {
			timeout = time.Duration(cpuParam.TimeOut)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		val, err := json.Marshal(cpuParam)
		if err != nil {
			logrus.Errorf("json.Marshal-failed", err.Error())
			return nil
		}
		uid := os.Getpid()
		uidStr := strconv.FormatInt(int64(uid), 10)
		t1 := time.Now().Format(time.RFC3339Nano)
		commandModel := &data.ExperimentModel{
			Uid:        uidStr,
			Command:    "create cpu fullload",
			CmdType:    "cpu",
			Status:     Created,
			Flag:       string(val),
			Error:      "",
			CreateTime: t1,
			UpdateTime: t1,
		}
		checkError(GetDS().InsertExperimentModel(commandModel))
		tools.Go(context.Background(), func(ctx context.Context) func(ctx context.Context, closing <-chan struct{}) {
			return func(ctx context.Context, closing <-chan struct{}) {
				if err := start(ctx, cpuParam.CpuCount, cpuParam.CpuPercent); err != nil {
					logrus.Errorf("start cpu failed", err.Error())
					return
				}
			}
		}(ctx))

		return transport.ReturnSuccessWithResult(uidStr)
	default:
	}
	return nil
}

// start burn cpu
func start(ctx context.Context, cpuCount, cpuPercent int) error {
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
		go burn(ctx, quota, slopePercent)
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

func burn(ctx context.Context, quota <-chan int64, slopePercent float64) {
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

			if (time.Now().Second() - t2) > 5 {
				logrus.Info("cpu-ending", time.Now().Second()-t2)
				fmt.Println("cpu-ending", time.Now().Second()-t2)
				return
			}
		case <-ctx.Done():
			times, ok := ctx.Deadline()
			uid := os.Getpid()
			uidStr := strconv.FormatInt(int64(uid), 10)
			checkError(GetDS().UpdateExperimentModelByUid(uidStr, Success, ""))
			logrus.Info("cpu演练时间结束啦", times, ok)
			fmt.Println("cpu演练时间结束啦", times, ok, time.Now().Second()-t2)
			return
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
