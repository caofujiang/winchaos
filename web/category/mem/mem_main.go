package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/data"
	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	modeStr := os.Args[1]
	memPercentStr := os.Args[2]
	uid := os.Args[3]
	memPercent, _ := strconv.Atoi(memPercentStr)
	pid := os.Getpid()
	pidStr := strconv.FormatInt(int64(pid), 10)
	val := &MemParams{
		Mode:       modeStr,
		MemPercent: memPercent,
		PID:        pidStr,
		UID:        uid,
	}

	result, err := json.Marshal(val)
	if err != nil {
		logrus.Errorf("memParam-json.Marshal-failed", err.Error())
		return
	}
	logrus.Info("cpu-main-param", uid, string(result))
	t1 := time.Now().Format(time.RFC3339Nano)
	commandModel := &data.ExperimentModel{
		Uid:        uid,
		Command:    "create mem load",
		CmdType:    "mem",
		Status:     CreatedMem,
		Flag:       string(result),
		Error:      "",
		CreateTime: t1,
		UpdateTime: t1,
	}
	checkMemError(GetMemDS().InsertExperimentModel(commandModel))
	tools.Go(context.Background(), func(ctx context.Context) func(ctx context.Context, closing <-chan struct{}) {
		return func(ctx context.Context, closing <-chan struct{}) {
			var memReserve, memRate int
			starts(ctx, memPercent, memReserve, memRate, modeStr, uid)
		}
	}(ctx))
	tools.Wait()
}

type MemParams struct {
	Mode       string `json:"mode"`
	MemPercent int    `json:"mem-percent"`
	Timeout    int    `json:"timeout"`
	PID        string `json:"pid"`
	UID        string `json:"uid"`
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
			checkMemError(GetMemDS().UpdateExperimentModelByUid(uid, SuccessMem, ""))
			logrus.Info("cpu-burn-done", uid)
			os.Exit(0)
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

// checkMemError for db operation
func checkMemError(err error) {
	if err != nil {
		log.Println(context.Background(), err.Error())
	}
}

// sqlite
var dsMem data.SourceI

// GetMemDS returns dataSource
func GetMemDS() data.SourceI {
	if dsMem == nil {
		dsMem = data.GetSource()
	}
	return dsMem
}

const (
	CreatedMem = "Created"
	SuccessMem = "Success"
)
