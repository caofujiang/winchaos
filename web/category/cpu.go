package category

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/transport"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
)

type Cpuparam struct {
	Cbt        ChaosbladeType
	Cmt        ChaosbladeCPUType
	CpuCount   int `json:"cpuCount"`
	CpuPercent int `json:"cpuPercent"`
}

func CpuResolver(ctx context.Context, cpuParam *Cpuparam) (response *transport.Response) {
	if cpuParam == nil {
		logrus.Errorf("cpuParam is nil")
		return nil
	}
	switch cpuParam.Cmt {
	case ChaosbladeTypeCPUFullLoad:
		if cpuParam == nil {
			logrus.Errorf("cpuParam cannot nil")
			return nil
		}
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
		uid := os.Getpid()
		start(ctx, cpuParam.CpuCount, cpuParam.CpuPercent)
		uidStr := strconv.FormatInt(int64(uid), 10)
		return transport.ReturnSuccessWithResult(uidStr)
	default:
	}
	return nil
}

// start burn cpu
func start(ctx context.Context, cpuCount, cpuPercent int) *spec.Response {
	ctx = context.WithValue(ctx, "cpuCount", cpuCount)
	runtime.GOMAXPROCS(cpuCount)
	logrus.Debugf("cpu counts: %d", cpuCount)
	slopePercent := float64(cpuPercent)
	climbTime := 5

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
	logrus.Errorf("cpu usage: %f", used)
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
				fmt.Println("time ending1======", time.Now().Second()-t2)
				break
			}
		case <-ctx.Done():
			times, ok := ctx.Deadline()
			fmt.Println("cpu演练时间结束啦！---------------", times, ok)
			break
		default:
			for time.Now().UnixNano()-startTime < q {
			}
			runtime.Gosched()
			time.Sleep(s)
			if (time.Now().Second() - t2) > 5 {
				fmt.Println("time ending2======", time.Now().Second()-t2)
				break
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
