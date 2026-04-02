package utils

import (
	"ekhoes-server/module"
	"log"
)

func Log(m module.Module, format string, a ...any) {
	log.Printf("[%s] %s", m.Id, format)
}
