package cmdexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

type CreateCommand struct {
}

func (ch *CreateCommand) Script(subCmd, downloadUrl string, args []string, timeout string) (response *transport.Response) {
	subCmdType := category.ChaosbladeScriptType(subCmd)
	switch subCmdType {
	case category.ChaosbladeScriptTypeExecute:
		uid, err := ch.execScript(downloadUrl, args, timeout)
		if err != nil {
			logrus.Warningf("exec Script:%s ,failed : %s ", downloadUrl, err.Error())
			return transport.ReturnFail(transport.ScriptFileExecuteWrong, err.Error())
		}
		return transport.ReturnSuccessWithResult(uid)

	case category.ChaosbladeScriptTypeExit:

	}
	return transport.ReturnFail(transport.ScriptFileExecuteWrong, "Script subCmdType error")
}

func (ch *CreateCommand) execScript(downloadUrl string, args []string, timeout string) (uid string, err error) {
	currentPath, err := os.Getwd()
	if err != nil {
		logrus.Warningf("os.Getwd error : %s ", err.Error())
		return "", err
	}
	scriptFileName := path.Base(downloadUrl)
	filePath := currentPath + "/" + scriptFileName
	err = downloadFile(downloadUrl, filePath)
	if err != nil {
		logrus.Warningf("download scriptFile  failed : %s ", err.Error())
		return "", err
	}
	argsStr := strings.Join(args, ",")
	surfix := path.Ext(scriptFileName)

	uid, err = ch.generateUid()
	if err != nil {
		return "", err
	}
	t1 := time.Now().Format(time.RFC3339Nano)
	if err != nil {
		return "", err
	}
	commandModel := &data.ExperimentModel{
		Uid:        uid,
		Command:    "create script execute",
		CmdType:    "script",
		SubCommand: filePath,
		Flag:       argsStr,
		Status:     Created,
		Error:      "",
		CreateTime: t1,
		UpdateTime: t1,
	}
	checkError(GetDS().InsertExperimentModel(commandModel))
	var cmd *exec.Cmd
	if surfix == ".bat" {
		cmd = exec.Command("cmd.exe", "/C", filePath, argsStr)
	} else if surfix == ".ps1" {
		// PowerShell 命令和参数
		cmdArgs := []string{
			"-ExecutionPolicy", "RemoteSigned",
			"-File", filePath,
		}
		cmdArgs = append(cmdArgs, args...)
		cmd = exec.Command("powershell.exe", cmdArgs...)
	} else {
		logrus.Warningf("file : %s  , format is wrong,please check !", filePath)
		return "", errors.New(filePath + " : format is wrong!")
	}

	timeStart := time.Now()
	result, err := cmd.CombinedOutput()
	if err != nil {
		checkError(GetDS().UpdateExperimentModelByUid(uid, Error, err.Error()))
		logrus.Warningf("cmd.CombinedOutput err  : %s!", err.Error())
		return "", err
	}
	//超时处理
	tt, err := strconv.ParseUint(timeout, 10, 64)
	if err != nil {
		timeDuartion, _ := time.ParseDuration(timeout)
		tt = uint64(timeDuartion.Seconds())
	}
	scriptExecteTime := time.Since(timeStart).Seconds()
	if tt > 0 && uint64(scriptExecteTime) > tt {
		cmd.Process.Kill()
	}
	checkError(GetDS().UpdateExperimentModelByUid(uid, Success, ""))
	//记录执行结果
	logrus.Infof("cmd.CombinedOutput result: %s!", string(result))
	fmt.Println("cmd.CombinedOutput result:", string(result))
	return uid, nil
}

// checkError for db operation
func checkError(err error) {
	if err != nil {
		log.Println(context.Background(), err.Error())
	}
}

func parseCommandPath(commandPath string) (string, string, error) {
	// chaosbd create docker cpu fullload
	cmds := strings.SplitN(commandPath, " ", 4)
	if len(cmds) < 4 {
		return "", "", fmt.Errorf("not illegal command")
	}
	return cmds[2], cmds[3], nil
}

func (ch *CreateCommand) generateUid() (string, error) {
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
	return ch.generateUid()
}

func downloadFile(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	// copy stream
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
