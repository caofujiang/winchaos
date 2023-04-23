package cmdexec

import (
	"fmt"
	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/transport"
	"github.com/sirupsen/logrus"
)

const (
	Created   = "Created"
	Success   = "Success"
	Error     = "Error"
	Destroyed = "Destroyed"
)

type StatusCommand struct {
	CommandType string
	//target      string
	//action      string
	//flag        string
	Uid    string
	Limit  string
	Status string
	Asc    string
}

// sqlite
var ds data.SourceI

// GetDS returns dataSource
func GetDS() data.SourceI {
	if ds == nil {
		ds = data.GetSource()
	}
	return ds
}

/*
  search  Experiments  result
*/

func (sc *StatusCommand) SearchExperiments(params *StatusCommand) (response *transport.Response) {
	if params.Uid != "" {
		result, err := GetDS().QueryExperimentModelByUid(params.Uid)
		fmt.Println("result", result)
		if err != nil {
			logrus.Errorf("GetDS().QueryExperimentModelByUid  error: %s ", err.Error())
			return transport.ReturnFail(transport.GetStatusExecuteWrong, err.Error())
		}
		if result == nil {
			return transport.ReturnFail(transport.HandlerNotFound, "not found")
		}
		return transport.ReturnSuccessWithResult(result)
	} else {
		var asc bool
		if params.Asc == "" || params.Asc == "true" {
			asc = true
		} else {
			asc = false
		}
		if params.Status == "" {
			params.Status = Success
		}
		result, err := GetDS().QueryExperimentModels(params.Status, params.Limit, asc)
		if err != nil {
			logrus.Errorf("GetDS().QueryExperimentModels  error: %s ", err.Error())
			return transport.ReturnFail(transport.GetStatusExecuteWrong, err.Error())
		}
		if result == nil {
			return transport.ReturnFail(transport.HandlerNotFound, "not found")
		}
		return transport.ReturnSuccessWithResult(result)
	}
}
