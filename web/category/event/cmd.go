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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	eventType := os.Args[4]
	eventCategory := category.ChaosbladeType(eventType)
	switch eventCategory {
	case category.ChaosbladeTypeCPU:
		// blade create cpu load --cpu-percent 60 --cpu-count 2
		cpuCountStr := os.Args[1]
		cpuPercentStr := os.Args[2]
		uid := os.Args[3]
		cpuCount, _ := strconv.Atoi(cpuCountStr)
		cpuPercent, _ := strconv.Atoi(cpuPercentStr)
		category.CpuRun(ctx, cpuCount, cpuPercent, uid)
	case category.ChaosbladeTypeMemory:
		modeStr := os.Args[1]
		memPercentStr := os.Args[2]
		uid := os.Args[3]
		memPercent, _ := strconv.Atoi(memPercentStr)
		category.MemRun(ctx, memPercent, modeStr, uid)
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
