package cmdexec

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
)

type Cpuparam struct {
	Cmt        category.ChaosbladeCPUType
	CpuCount   int    `json:"cpu-count"`
	CpuPercent int    `json:"cpu-percent"`
	TimeOut    int    `json:"timeOut"`
	PID        string `json:"pid"`
}

func CpuResolver(cpuParam *Cpuparam) (response *transport.Response) {
	if cpuParam == nil {
		logrus.Errorf("cpuParam is nil")
		return transport.ReturnFail(transport.ParameterTypeError, cpuParam)
	}
	switch cpuParam.Cmt {
	case category.ChaosbladeTypeCPUFullLoad:
		if cpuParam.CpuPercent != 0 {
			if cpuParam.CpuPercent > 100 || cpuParam.CpuPercent < 0 {
				logrus.Errorf("`%d`: cpu-percent is illegal, it must be a positive integer and not bigger than 100", cpuParam.CpuPercent)
				return transport.ReturnFail(transport.ParameterTypeError, cpuParam.CpuPercent)
			}
		}
		if cpuParam.CpuCount != 0 {
			if cpuParam.CpuCount <= 0 || cpuParam.CpuCount > runtime.NumCPU() {
				cpuParam.CpuCount = runtime.NumCPU()
			}
		} else {
			cpuParam.CpuCount = 1
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

		uid, err := cpuParam.generateUid()
		if err != nil {
			logrus.Errorf("cpuParam.generateUid-failed", err.Error())
			return transport.ReturnFail(transport.ParameterTypeError, err.Error())
		}

		tools.Go(context.Background(), func(ctx context.Context) func(ctx context.Context, closing <-chan struct{}) {
			return func(ctx context.Context, closing <-chan struct{}) {
				currentPath, err := os.Getwd()
				if err != nil {
					logrus.Warningf("os.Getwd error : %s ", err.Error())
					return
				}
				path := currentPath + "/" + "cpu.exe"
				cmd := exec.Command(path, strconv.Itoa(cpuParam.CpuCount), strconv.Itoa(cpuParam.CpuPercent), uid)
				output, err := cmd.Output()
				if err != nil {
					logrus.Errorf("cmd.Output-failed", err.Error(), string(output))
					return
				}
				return
			}
		}(ctx))
		return transport.ReturnSuccessWithResult(uid)
	default:
	}
	return response
}

func (cpm *Cpuparam) generateUid() (string, error) {
	uid, err := util.GenerateUid()
	if err != nil {
		return "", err
	}
	model, err := GetDS().QueryExperimentModelByUid(uid)
	if err != nil {
		return "", err
	}
	if model == nil {
		return uid, nil
	}
	return cpm.generateUid()
}
