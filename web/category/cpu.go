package category

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
)

type Cpuparam struct {
	Cbt        ChaosbladeType
	Cmt        ChaosbladeCPUType
	UID        string `json:"uid"`
	CpuCount   int    `json:"cpuCount"`
	CpuPercent int    `json:"cpuPercent"`
}

func CpuResolver(ctx context.Context, cpuParam *Cpuparam) {
	if cpuParam == nil {
		logrus.Errorf("cpuParam is nil")
		return
	}
	switch cpuParam.Cmt {
	case ChaosbladeTypeCPUFullLoad:
		if err := CpuLoadExec(ctx, cpuParam); err != nil {
			logrus.Errorf("CpuLoadExec failed", err.Error())
			return
		}
	default:
	}
}

func CpuLoadExec(ctx context.Context, cpuParam *Cpuparam) error {
	// TODO destroy
	//if _, ok := stop(ctx); ok {
	//return ce.stop(ctx)
	//}
	if cpuParam == nil {
		logrus.Errorf("cpuParam cannot nil")
		return errors.New("cpuParam is nil")
	}

	if cpuParam.CpuPercent != 0 {
		if cpuParam.CpuPercent > 100 || cpuParam.CpuPercent < 0 {
			logrus.Errorf("`%d`: cpu-percent is illegal, it must be a positive integer and not bigger than 100", cpuParam.CpuPercent)
			return fmt.Errorf("cpuPercent is nil ,%d", cpuParam.CpuPercent)
		}
	}

	if cpuParam.CpuCount != 0 {
		if cpuParam.CpuCount <= 0 || cpuParam.CpuCount > runtime.NumCPU() {
			cpuParam.CpuCount = runtime.NumCPU()
		}
	}

	resp := start(ctx, cpuParam.CpuCount, cpuParam.CpuPercent)
	if resp.Code == 200 {
		logrus.Errorf("start response success")
		return nil
	}
	return errors.New("resp not success")
}

// start burn cpu
func start(ctx context.Context, cpuCount, cpuPercent int) *spec.Response {
	ctx = context.WithValue(ctx, "cpuCount", cpuCount)
	runtime.GOMAXPROCS(cpuCount)
	log.Debugf(ctx, "cpu counts: %d", cpuCount)
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
	log.Debugf(ctx, "cpu usage: %f", used)
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
			s, _ = time.ParseDuration(strconv.FormatInt(ds, 10) + "ns")
			fmt.Println("startTime++++++++++++++++++++++++", time.Now().Second()-t2)
			if (time.Now().Second() - t2) > 5 {
				fmt.Println("time ending1======", time.Now().Second()-t2)
				break
			}
		case <-ctx.Done():
			times, ok := ctx.Deadline()
			fmt.Println("cpu演练时间结束啦！---------------", times, ok)
			break
		default:
			fmt.Println("default+++++++++++++++++++", s)
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

// stop burn cpu
func stop(ctx context.Context) *spec.Response {
	// TODO 根据Uid 销毁
	//if uid != ""
	//ctx = context.WithValue(ctx, "bin", BurnCpuBin)
	//return exec.Destroy(ctx, ce.channel, "cpu fullload")
	return nil
}

func getUsed(ctx context.Context) float64 {
	totalCpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Fatalf(ctx, "get cpu usage fail, %s", err.Error())
	}
	return totalCpuPercent[0]
}
