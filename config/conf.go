package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/host"
)

var name string = "Ekhoes API server"
var version string = "1.0.0-alpha"
var buildTime string

func Init() {

	// Load .env (as environment variables)
	if err := godotenv.Load(); err != nil {
		//fmt.Println(".env not found, continue anyway...")
	}

	Runtime.StartTime = time.Now().UTC()
	Runtime.Database = "None"
	Runtime.Cache = "None"

	hostInfo, _ := host.Info()

	if os.Getenv("EKHOES_INSTANCE_NAME") == "" {
		os.Setenv("EKHOES_INSTANCE_NAME", "EKHOES-"+hostInfo.Hostname)
	}
}

func Name() string {
	return name
}

func Version() string {
	return version
}

func BuildTime() string {
	return buildTime
}

func PosgresEnabled() bool {
	return os.Getenv("EKHOES_DB_ENABLED") == "true"
}

func RedisEnabled() bool {
	return os.Getenv("EKHOES_REDIS_ENABLED") == "true"
}

func Port() int {
	port := 9876

	if os.Getenv("EKHOES_PORT") != "" {
		port, _ = strconv.Atoi(os.Getenv("EKHOES_PORT"))

	}

	return port
}

func SetPort(port int) {
	os.Setenv("PORT", strconv.Itoa(port))
}

func JWTSecret() string {
	return os.Getenv("EKHOES_JWT_SECRET")
}

func InstanceName() string {
	return os.Getenv("EKHOES_INSTANCE_NAME")
}
