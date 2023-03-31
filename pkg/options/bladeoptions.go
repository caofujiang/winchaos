package options

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/caofujiang/winchaos/pkg/bash"
	"github.com/caofujiang/winchaos/pkg/tools"
)

const (
	BladeBin            = "blade"
	BladeDirName        = "chaosblade"
	BladeDatFileName    = "chaosblade.dat"
	BladeBakDatFileName = "chaosblade.dat.bak"
	CtlForChaos         = "chaosctl.sh"
)

// 为解决 Unable to access jarfile /root/chaos/chaosblade/lib/sandbox/lib/sandbox-core.jar
// 需要将 chaosblade 部署到 C:下
//var BladeHome = path.Join("/opt", BladeDirName)

//var BladeBinPath = path.Join(BladeHome, BladeBin)

var BladeBinPath = func() string {
	var (
		BladeHome, bladeBinPath string
	)

	if tools.IsWindows() {
		BladeHome = path.Join("C:\\", BladeDirName)
	} else {
		BladeHome = path.Join("/opt", BladeDirName)
	}
	bladeBinPath = path.Join(BladeHome, BladeBin)
	return bladeBinPath
}

var CtlPathFunc = func() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.Warning("get current directory failed")
		return ""
	}

	if tools.IsExist(path.Join(dir, CtlForChaos)) {
		return path.Join(dir, CtlForChaos)
	}
	return ""
}

// GetChaosBladeVersion
func GetChaosBladeVersion() (string, error) {
	if !tools.IsExist(BladeBinPath()) {
		return "", fmt.Errorf("blade bin file not exist")
	}

	result, errMsg, isSuccess := bash.ExecScript(context.TODO(), BladeBinPath(), "version")
	if !isSuccess {
		return "", fmt.Errorf(errMsg)
	}

	versionInfos := strings.Split(strings.TrimSpace(result), "\n")
	if len(versionInfos) == 0 {
		return "", fmt.Errorf("cannot get blade version")
	}

	versionInfo := versionInfos[0]
	hasPrefix := strings.HasPrefix(versionInfo, "version")
	if !hasPrefix {
		return "", fmt.Errorf("cannot get version info from first line. %s", result)
	}
	versionArr := strings.Split(versionInfo, ":")
	if len(versionArr) != 2 {
		return "", fmt.Errorf("parse version info error. %s", versionInfo)
	}
	version := strings.TrimSpace(versionArr[1])
	logrus.Infof("ChaosBlade version is %s", version)
	return version, nil
}
