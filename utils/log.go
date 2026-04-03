package utils

import (
	"ekhoes-server/module"
	"fmt"
	"log"
)

func Log(m module.Module, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	log.Printf("[%s] %s", m.Id, msg)
}
