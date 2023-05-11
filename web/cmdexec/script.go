package cmdexec

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
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
		logrus.Errorf("os.Getwd error : %s ", err.Error())
		return "", err
	}
	scriptFileName := path.Base(downloadUrl)
	tarFilePathDir := currentPath + "/" + strconv.FormatInt(time.Now().Unix(), 10) + "/"
	err = isExistDir(tarFilePathDir)
	if err != nil {
		logrus.Errorf("create tar FilePath Directory  failed : %s ", err.Error())
		return "", err
	}
	filePath := tarFilePathDir + scriptFileName
	err = downloadFile(downloadUrl, filePath)
	if err != nil {
		logrus.Errorf("download scriptFile  failed : %s ", err.Error())
		return "", err
	}
	err = unTar(filePath, tarFilePathDir)
	if err != nil {
		logrus.Errorf("unTar scriptFile  failed : %s ", err.Error())
		return "", err
	}

	filename, fileType, err := listDir(tarFilePathDir)
	if err != nil {
		fmt.Println("err: ", err.Error())
	}
	if filename == "" || fileType == "" {
		logrus.Errorf("unTared scriptFiles  not exist main file")
		return "", errors.New("unTared scriptFiles  not exist main file")
	}
	filePath = tarFilePathDir + filename

	argsStr := strings.Join(args, ",")
	//surfix := path.Ext(scriptFileName)
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
		SubCommand: tarFilePathDir,
		Flag:       argsStr,
		Status:     Created,
		Error:      "",
		CreateTime: t1,
		UpdateTime: t1,
	}
	checkError(GetDS().InsertExperimentModel(commandModel))
	var cmd *exec.Cmd
	if fileType == ".bat" {
		cmd = exec.Command("cmd.exe", "/C", filePath, argsStr)
	} else if fileType == ".ps1" {
		// PowerShell 命令和参数
		cmdArgs := []string{
			"-ExecutionPolicy", "RemoteSigned",
			"-File", filePath,
		}
		cmdArgs = append(cmdArgs, args...)
		cmd = exec.Command("powershell.exe", cmdArgs...)
	} else {
		logrus.Errorf("file : %s  , format is wrong,please check !", filePath)
		return "", errors.New(filePath + " : format is wrong!")
	}

	timeStart := time.Now()
	result, err := cmd.CombinedOutput()
	if err != nil {
		checkError(GetDS().UpdateExperimentModelByUid(uid, Error, err.Error()))
		logrus.Errorf("cmd.CombinedOutput err  : %s!", err.Error())
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

// 解压到目录
func unTar(filename, tarFilePathDir string) error {
	// 打开tar文件
	f, err := os.Open(filename)
	if err != nil {
		logrus.Errorf("func unTar os.Open(%s)  err  : %s!", filename, err.Error())
		return err
	}
	defer f.Close()
	// 创建一个Reader
	reader := tar.NewReader(f)
	for {
		// 读取每一块内容
		hdr, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// 创建对应的目录
		os.MkdirAll(filepath.Dir(hdr.Name), 0666)
		// 创建tar归档中的文件
		f, err := os.OpenFile(tarFilePathDir+hdr.Name, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		defer f.Close()
		// 写入文件中
		_, err = io.Copy(f, reader)
		if err != nil {
			return err
		}
	}
	return nil
}

// 不存在就创建
func isExistDir(dirpathdir string) error {
	_, e := os.Stat(dirpathdir)
	if e != nil {
		if os.IsNotExist(e) {
			if e := os.MkdirAll(dirpathdir, os.ModePerm); e != nil {
				logrus.Errorf("func isExistDir os.MkdirAll err  : %v\n%s", e, debug.Stack())
				return e
			}
		} else {
			return e
		}
	}
	return nil
}

func listDir(dirname string) (filename, fileType string, err error) {
	infos, err := ioutil.ReadDir(dirname)
	if err != nil {
		return "", "", err
	}
	for _, info := range infos {
		filename := filepath.Base(info.Name())
		//获取文件的后缀(文件类型)
		fileType = path.Ext(filename)
		//获取文件名称(不带后缀)
		fileNameOnly := strings.TrimSuffix(filename, fileType)
		if fileNameOnly == "main" {
			return filename, fileType, nil
		}
	}
	return "", "", nil
}
