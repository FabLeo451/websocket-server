package config

import (
	"time"
)

type RuntimeStruct struct {
	StartTime    time.Time
	InstanceName string
	Database     string
	Local        bool
	Cache        string
}

var Runtime RuntimeStruct

func Local() bool {
	return Runtime.Local
}
