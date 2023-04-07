package category

import (
	"context"
	"errors"
	"fmt"
	"os"
	os_exec "os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chaosblade-io/chaosblade-spec-go/log"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/shirou/gopsutil/cpu"
	"github.com/sirupsen/logrus"
)

type Cpuparam struct {
	Cbt        ChaosbladeType
	Cmt        ChaosbladeCPUType
	CpuCount   int    `json:"cpuCount"`
	CpuList    string `json:"cpuList"`
	CpuPercent int    `json:"cpuPercent"`
	ClimbTime  int    `json:"climbTime"`
	CpuIndex   string `json:"cpuIndex"`
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
	if _, ok := spec.IsDestroy(ctx); ok {
		//return ce.stop(ctx)
	}

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

	if cpuParam.ClimbTime != 0 {
		if cpuParam.ClimbTime > 600 || cpuParam.ClimbTime < 0 {
			log.Errorf(ctx, "`%d`: climb-time is illegal, climb-time value must be a positive integer and not bigger than 600", cpuParam.ClimbTime)
			return errors.New("climb-time must be a positive integer and not bigger than 600")
		}
	}

	resp := start(ctx, cpuParam.CpuList, cpuParam.CpuCount, cpuParam.CpuPercent, cpuParam.ClimbTime, cpuParam.CpuIndex)
	if resp.Code == 200 {
		logrus.Errorf("start response success")
		return nil
	}
	return errors.New("resp not success")
}

// start burn cpu
func start(ctx context.Context, cpuList string, cpuCount, cpuPercent, climbTime int, cpuIndexStr string) *spec.Response {
	ctx = context.WithValue(ctx, "cpuCount", cpuCount)
	if cpuList != "" {
		cores, err := util.ParseIntegerListToStringSlice("cpu-list", cpuList)
		if err != nil {
			logrus.Errorf("`%s`: cpu-list is illegal, %s", cpuList, err.Error())
			return spec.ResponseFailWithFlags(spec.ParameterIllegal, "cpu-list", cpuList, err.Error())
		}
		for _, core := range cores {
			args := fmt.Sprintf(`%s create cpu fullload --cpu-count 1 --cpu-percent %d --climb-time %d --cpu-index %s --uid %s`, os.Args[0], cpuPercent, climbTime, core, ctx.Value(spec.Uid))
			args = fmt.Sprintf("-c %s %s", core, args)
			argsArray := strings.Split(args, " ")
			command := os_exec.CommandContext(ctx, "taskset", argsArray...)
			command.SysProcAttr = &syscall.SysProcAttr{}
			if err := command.Start(); err != nil {
				return spec.ReturnFail(spec.OsCmdExecFailed, fmt.Sprintf("taskset exec failed, %v", err))
			}
		}
		return spec.ReturnSuccess(ctx.Value(spec.Uid))
	}

	runtime.GOMAXPROCS(cpuCount)
	log.Debugf(ctx, "cpu counts: %d", cpuCount)
	slopePercent := float64(cpuPercent)

	var cpuIndex int
	percpu := false
	if cpuIndexStr != "" {
		percpu = true
		var err error
		cpuIndex, err = strconv.Atoi(cpuIndexStr)
		if err != nil {
			logrus.Errorf("`%s`: cpu-index is illegal, cpu-index value must be a positive integer", cpuIndexStr)
			return spec.ResponseFailWithFlags(spec.ParameterIllegal, "cpu-index", cpuIndexStr, "it must be a positive integer")
		}
	}

	// make CPU slowly climb to some level, to simulate slow resource competition
	// which system faults cannot be quickly noticed by monitoring system.
	slope(ctx, cpuPercent, climbTime, &slopePercent, percpu, cpuIndex)

	quota := make(chan int64, cpuCount)
	for i := 0; i < cpuCount; i++ {
		go burn(ctx, quota, slopePercent, percpu, cpuIndex)
	}
	for {
		q := getQuota(ctx, slopePercent, percpu, cpuIndex)
		for i := 0; i < cpuCount; i++ {
			quota <- q
		}
	}
}

const BurnCpuBin = "chaos_burncpu"

type cpuExecutor struct {
	//channel spec.Channel
}

func (ce *cpuExecutor) Name() string {
	return "cpu"
}

func (ce *cpuExecutor) SetChannel(channel spec.Channel) {
	//ce.channel = channel
}

const period = int64(1000000000)

func slope(ctx context.Context, cpuPercent int, climbTime int, slopePercent *float64, percpu bool, cpuIndex int) {
	if climbTime != 0 {
		var ticker = time.NewTicker(time.Second)
		*slopePercent = getUsed(ctx, percpu, cpuIndex)
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

func getQuota(ctx context.Context, slopePercent float64, percpu bool, cpuIndex int) int64 {
	used := getUsed(ctx, percpu, cpuIndex)
	log.Debugf(ctx, "cpu usage: %f , percpu: %v, cpuIndex %d", used, percpu, cpuIndex)
	dx := (slopePercent - used) / 100
	busy := int64(dx * float64(period))
	return busy
}

func burn(ctx context.Context, quota <-chan int64, slopePercent float64, percpu bool, cpuIndex int) {
	q := getQuota(ctx, slopePercent, percpu, cpuIndex)
	ds := period - q
	if ds < 0 {
		ds = 0
	}
	s, _ := time.ParseDuration(strconv.FormatInt(ds, 10) + "ns")
	for {
		startTime := time.Now().UnixNano()
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
		default:
			for time.Now().UnixNano()-startTime < q {
			}
			runtime.Gosched()
			time.Sleep(s)
		}
	}
}

// stop burn cpu
func (ce *cpuExecutor) stop(ctx context.Context) *spec.Response {
	//ctx = context.WithValue(ctx, "bin", BurnCpuBin)
	//return exec.Destroy(ctx, ce.channel, "cpu fullload")
	return nil
}

func getUsed(ctx context.Context, percpu bool, cpuIndex int) float64 {
	totalCpuPercent, err := cpu.Percent(time.Second, percpu)
	if err != nil {
		log.Fatalf(ctx, "get cpu usage fail, %s", err.Error())
	}
	if percpu {
		if cpuIndex > len(totalCpuPercent) {
			log.Fatalf(ctx, "illegal cpu index %d", cpuIndex)
		}
		return totalCpuPercent[cpuIndex]
	}
	return totalCpuPercent[0]
}
