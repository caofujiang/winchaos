package cmdexec

import (
	"fmt"
	"os"
	"strconv"

	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/sirupsen/logrus"
)

type DestroyCommand struct {
}

func DestroyExperiment(uid string) (response *transport.Response) {
	//根据uid查库，查询执行命令，更加不同的命令类型，执行不同的 destroy 实现
	experimentModel, err := GetDS().QueryExperimentModelByUid(uid)
	if err != nil {
		logrus.Warningf("GetDS().QueryExperimentModelByUid,error : %s ", err.Error())
		return transport.ReturnFail(transport.DestroyedExperimentNotFound, uid, err.Error())
	}
	if experimentModel.Status == Destroyed {
		return transport.ReturnSuccessWithResult("Experiment already destroyed")
	}
	cmdType := category.ChaosbladeType(experimentModel.CmdType)
	switch cmdType {
	case category.ChaosbladeTypeCPU:
		if err := CPUDestroy(uid); err != nil {
			logrus.Warningf("destroy  the cpu error : %s ", err.Error())
			return transport.ReturnFail(transport.DestroyedExperimentError, err.Error())
		}
	case category.ChaosbladeTypeMemory:

	case category.ChaosbladeTypeScript:
		//清理本地的文件
		filePath := experimentModel.SubCommand
		fmt.Println("filePath   ", filePath)
		err := os.Remove(filePath)
		if err != nil {
			logrus.Warningf("destroy  script Experiment os.Remove error : %s ", err.Error())
			return transport.ReturnFail(transport.DestroyedExperimentError, err.Error())
		}
	}
	checkError(GetDS().UpdateExperimentModelByUid(uid, Destroyed, ""))
	return transport.ReturnSuccess()
}

// destroy burn cpu
func CPUDestroy(uid string) error {
	// TODO 根据Uid 销毁
	pid, err := strconv.Atoi(uid)
	if err != nil {
		logrus.Errorf("strconv.Atoi failed")
		return err
	}
	if handle, err := os.FindProcess(pid); err == nil {
		if err := handle.Kill(); err != nil {
			logrus.Errorf("the cpu process not kill", err.Error())
		}
	}
	return nil
}
