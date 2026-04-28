package utils

import (
	"ekhoes-server/common"
	"fmt"
	"log"
)

func Log(m common.Module, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	log.Printf("[%s] %s", m.Id, msg)
}

func Debug(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	log.Printf("[DEBUG] %s", msg)
}

func LogErr(m common.Module, err error) {
	Log(m, "Error: %s", err.Error())
}
