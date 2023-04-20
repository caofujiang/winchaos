package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/caofujiang/winchaos/conn"
	closer "github.com/caofujiang/winchaos/conn/close"
	"github.com/caofujiang/winchaos/conn/connect"
	"github.com/caofujiang/winchaos/conn/heartbeat"
	chaoshttp "github.com/caofujiang/winchaos/pkg/http"
	"github.com/caofujiang/winchaos/pkg/log"
	"github.com/caofujiang/winchaos/pkg/options"
	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/caofujiang/winchaos/transport"
	api2 "github.com/caofujiang/winchaos/web/api"
)

func main() {
	options.NewOptions()
	log.InitLog(&options.Opts.LogConfig)

	options.Opts.SetOthersByFlags()

	// new transport newConn
	clientInstance, err := chaoshttp.NewHttpClient(options.Opts.TransportConfig)
	if err != nil {
		logrus.Errorf("create transport client instance failed, err: %s", err.Error())
		handlerErr(err)
	}
	transportClient := transport.NewTransportClient(clientInstance)
	transport.InitTransprotUri()

	// conn to server
	connectClient := connect.NewClientConnectHandler(transportClient)
	heartbeatClient := heartbeat.NewClientHeartbeatHandler(options.Opts.HeartbeatConfig, transportClient)
	newConn := conn.NewConn()
	newConn.Register(transport.API_REGISTRY, connectClient)
	newConn.Register(transport.API_HEARTBEAT, heartbeatClient)
	newConn.Start()

	// new api
	api := api2.NewAPI()
	err = api.Register(transportClient)

	if err != nil {
		logrus.Errorf("register api failed, err: %s", err.Error())
		handlerErr(err)
	}

	// listen server
	go func() {
		defer tools.PanicPrintStack()
		err = http.ListenAndServe(":"+options.Opts.Port, nil)
		fmt.Println("http.ListenAndServe")
		if err != nil {
			logrus.Warningln("Start http server failed")
			handlerErr(err)
		}
	}()

	//handlerSuccess()

	closeClient := closer.NewClientCloseHandler(transportClient)
	tools.Hold(closeClient)

}

//func handlerSuccess() {
//	pid := os.Getpid()
//	err := writePid(pid)
//	if err != nil {
//		logrus.Panic("write pid: ", GetPidFile(), " failed. ", err)
//	}
//}

func handlerErr(err error) {
	if err == nil {
		return
	}
	logrus.Warningf("start agent failed because of %v", err)
	writePid(-1)
	logrus.Errorf("chaos agent will exit")
	os.Exit(1)
}

func writePid(pid int) error {
	file, err := os.OpenFile(GetPidFile(), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strconv.Itoa(pid))
	return err
}

func GetPidFile() string {
	var pidFile string
	if tools.IsWindows() {
		pidFile = "C:\\chaos.pid"
	} else if tools.IsUnix() {
		pidFile = "/var/run/chaos.pid"
	}
	return pidFile
}
