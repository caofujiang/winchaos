package cmdexec

import (
	"fmt"
	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/transport"
	"github.com/chaosblade-io/chaosblade-spec-go/spec"
	"github.com/chaosblade-io/chaosblade-spec-go/util"
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
			return transport.ReturnSuccessWithResult(struct{}{})
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
		return transport.ReturnSuccessWithResult(result)
	}
}

func (sc *StatusCommand) runStatus(args []string) error {
	var uid = ""
	if len(args) > 0 {
		uid = args[0]
	} else {
		uid = sc.Uid
	}
	var result interface{}
	var err error
	switch sc.CommandType {
	case "create", "destroy":
		if uid != "" {
			result, err = GetDS().QueryExperimentModelByUid(uid)
		} else {
			//result, err = GetDS().QueryExperimentModels(sc.target, sc.action, sc.flag, sc.status, sc.limit, sc.asc)
		}
	default:
		if uid == "" {
			return spec.ResponseFailWithFlags(spec.ParameterLess, "type|uid", "must specify the right type or uid")
		}
		result, err = GetDS().QueryExperimentModelByUid(uid)
		if util.IsNil(result) || err != nil {
			//result, err = GetDS().QueryPreparationByUid(uid)
		}
	}

	//response := spec.ReturnSuccess(result)
	return nil
}
