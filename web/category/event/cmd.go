package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caofujiang/winchaos/pkg/tools"
	"github.com/caofujiang/winchaos/web/category"
	"github.com/sirupsen/logrus"
)

func main() {
	eventType := os.Args[4]
	eventCategory := category.ChaosbladeType(eventType)

	timeoutStr := os.Args[5]
	timeout, _ := strconv.Atoi(timeoutStr)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	switch eventCategory {
	case category.ChaosbladeTypeCPU:
		// blade create cpu load --cpu-percent 60 --cpu-count 2
		cpuCountStr := os.Args[1]
		cpuPercentStr := os.Args[2]
		uid := os.Args[3]
		cpuCount, _ := strconv.Atoi(cpuCountStr)
		cpuPercent, _ := strconv.Atoi(cpuPercentStr)
		category.CpuRun(ctx, cpuCount, cpuPercent, uid, timeout)
	case category.ChaosbladeTypeMemory:
		modeStr := os.Args[1]
		memPercentStr := os.Args[2]
		uid := os.Args[3]
		memPercent, _ := strconv.Atoi(memPercentStr)
		category.MemRun(ctx, memPercent, modeStr, uid, timeout)
	default:
		logrus.Errorf("not expect eventCategory", eventCategory)
		return
	}
	tools.Wait()
	return
}

// checkError for db operation
func checkError(err error) {
	if err != nil {
		log.Println(context.Background(), err.Error())
	}
}
