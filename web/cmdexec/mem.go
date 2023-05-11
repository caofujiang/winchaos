package cmdexec

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
)

type MemParam struct {
	Cmt        category.ChaosbladeMemoryType
	Mode       string `json:"mode"`
	MemPercent int    `json:"mem-percent"`
	Timeout    int    `json:"timeout"`
	PID        string `json:"pid"`
}

func MemResolver(memParam *MemParam) (response *transport.Response) {
	if memParam == nil {
		logrus.Errorf("memParam is nil")
		return nil
	}
	switch memParam.Cmt {
	case category.ChaosbladeMemoryTypeLoad:
		memPercent := memParam.MemPercent
		if memPercent != 0 {
			if memPercent > 100 || memPercent < 0 {
				logrus.Errorf("`%d`: mem-percent  must be a positive integer and not bigger than 100", memParam.MemPercent)
				return transport.ReturnFail(transport.ParameterTypeError, memParam.MemPercent)
			}
		} else {
			memPercent = 100
		}

		if memParam.Mode == "" {
			memParam.Mode = "ram"
		}

		var timeout int
		if memParam.Timeout == 0 {
			// 默认超时
			timeout = 60
		} else {
			timeout = memParam.Timeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		uid, err := memParam.generateUid()
		if err != nil {
			logrus.Errorf("memParam.generateUid-failed", err.Error())
			return transport.ReturnFail(transport.ParameterTypeError, err.Error())
		}

		tools.Go(context.Background(), func(ctx context.Context) func(ctx context.Context, closing <-chan struct{}) {
			return func(ctx context.Context, closing <-chan struct{}) {
				currentPath, err := os.Getwd()
				if err != nil {
					logrus.Warningf("os.Getwd-mem-error : %s ", err.Error())
					return
				}
				path := currentPath + "/" + "os.exe"
				cmd := exec.Command(path, memParam.Mode, strconv.Itoa(memPercent), uid, "mem", strconv.Itoa(timeout))
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
	return transport.ReturnFail(transport.ParameterTypeError)
}

func (mp *MemParam) generateUid() (string, error) {
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
	return mp.generateUid()
}
