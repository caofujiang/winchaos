package cmdexec

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/caofujiang/winchaos/transport"
	"github.com/caofujiang/winchaos/web/category"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
)

type MemParam struct {
	Cmt        category.ChaosbladeMemoryType
	Mode       string `json:"mode"`
	MemPercent string `json:"memPercent"`
	TimeOut    int    `json:"timeOut"`
	UID        string `json:"uid"`
}

func MemResolver(memParam *MemParam) (response *transport.Response) {
	if memParam == nil {
		logrus.Errorf("memParam is nil")
		return nil
	}
	switch memParam.Cmt {
	case category.ChaosbladeMemoryTypeLoad:
		var timeout time.Duration
		if memParam.TimeOut == 0 {
			// 默认超时
			timeout = 60 * time.Second
		} else {
			timeout = time.Duration(memParam.TimeOut)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		val, err := json.Marshal(memParam)
		if err != nil {
			logrus.Errorf("json.Marshal-failed", err.Error())
			return nil
		}
		uid := os.Getpid()
		uidStr := strconv.FormatInt(int64(uid), 10)
		memParam.UID = uidStr
		t1 := time.Now().Format(time.RFC3339Nano)
		commandModel := &data.ExperimentModel{
			Uid:        uidStr,
			Command:    "create mem load",
			CmdType:    "mem",
			Status:     Created,
			Flag:       string(val),
			Error:      "",
			CreateTime: t1,
			UpdateTime: t1,
		}
		checkError(GetDS().InsertExperimentModel(commandModel))
		tools.Go(context.Background(), func(ctx context.Context) func(ctx context.Context, closing <-chan struct{}) {
			return func(ctx context.Context, closing <-chan struct{}) {
				if err := ExecMem(ctx, memParam); err != nil {
					logrus.Errorf("ExecMem failed", err.Error())
					return
				}
			}
		}(ctx))

		return transport.ReturnSuccessWithResult(uidStr)
	default:
	}
	return nil
}

func ExecMem(ctx context.Context, model *MemParam) error {
	var memPercent, memReserve, memRate int
	memPercentStr := model.MemPercent
	burnMemModeStr := model.Mode
	if memPercentStr != "" {
		var err error
		memPercent, err = strconv.Atoi(memPercentStr)
		if err != nil {
			logrus.Errorf("`%s`: mem-percent  must be a positive integer", memPercentStr)
			return err
		}
		if memPercent > 100 || memPercent < 0 {
			logrus.Errorf("`%s`: mem-percent  must be a positive integer and not bigger than 100", memPercentStr)
			return err
		}
	} else {
		memPercent = 100
	}
	starts(ctx, memPercent, memReserve, memRate, burnMemModeStr, model.UID)
	return nil
}

// start burn mem
func starts(ctx context.Context, memPercent, memReserve, memRate int, burnMemMode string, uid string) {
	var cache = make(map[int][]Block, 1)
	var count = 1
	cache[count] = make([]Block, 0)
	if memRate <= 0 {
		memRate = 100
	}
	includeBufferCache := true
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			checkError(GetDS().UpdateExperimentModelByUid(uid, Success, ""))
			fmt.Println("burn 超时了停止运行==========")
			return
		case <-ticker.C:
			_, expectMem, err := calculateMemSize(ctx, burnMemMode, memPercent, memReserve, includeBufferCache)
			if err != nil {
				logrus.Errorf("calculate memsize err, %v", err.Error())
			}
			fillMem := expectMem
			if expectMem > 0 {
				if expectMem > int64(memRate) {
					fillMem = int64(memRate)
				} else {
					fillMem = expectMem / 10
					if fillMem == 0 {
						continue
					}
				}
				fillSize := int(8 * fillMem)
				buf := cache[count]
				if cap(buf)-len(buf) < fillSize && int(math.Floor(float64(cap(buf))*1.25)) >= int(8*expectMem) {
					count += 1
					cache[count] = make([]Block, 0)
					buf = cache[count]
				}
				logrus.Debugf("count: %d, len(buf): %d, cap(buf): %d, expect mem: %d, fill size: %d", count, len(buf), cap(buf), expectMem, fillSize)
				cache[count] = append(buf, make([]Block, fillSize)...)
			}
		}
	}
}

// 128K
type Block [32 * 1024]int32

func calculateMemSize(ctx context.Context, burnMemMode string, percent, reserve int, includeBufferCache bool) (int64, int64, error) {
	total, available, err := getAvailableAndTotal(ctx, burnMemMode, includeBufferCache)
	if err != nil {
		return 0, 0, err
	}
	reserved := int64(0)
	if percent != 0 {
		reserved = (total * int64(100-percent) / 100) / 1024 / 1024
	} else {
		reserved = int64(reserve)
	}
	expectSize := available/1024/1024 - reserved
	logrus.Debugf("available: %d, percent: %d, reserved: %d, expectSize: %d", available/1024/1024, percent, reserved, expectSize)

	return total / 1024 / 1024, expectSize, nil
}

func getAvailableAndTotal(ctx context.Context, burnMemMode string, includeBufferCache bool) (int64, int64, error) {
	//no limit
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, err
	}
	total := int64(virtualMemory.Total)
	available := int64(virtualMemory.Free)
	if burnMemMode == "ram" && !includeBufferCache {
		available = available + int64(virtualMemory.Buffers+virtualMemory.Cached)
	}
	return total, available, nil
}
