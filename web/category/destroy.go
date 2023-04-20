package category

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

// destroy burn cpu
func Destroy(uid string) error {
	// TODO 根据Uid 销毁
	pid, err := strconv.Atoi(uid)
	if err != nil {
		logrus.Errorf("strconv.Atoi failed")
		return err
	}
	if handle, err := os.FindProcess(pid); err == nil {
		if err := handle.Kill(); err != nil {
			logrus.Errorf("the cpu process not kill", err.Error())
		}
	}
	return nil
}
