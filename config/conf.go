package config

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/spf13/viper"
)

var name string = "Ekhoes API server"
var version string = "1.0.0"
var buildTime string

type Configuration struct {
	Package struct {
		Name      string
		Version   string
		BuildTime string
	}
	DB struct {
		Enabled   bool
		Host      string
		Port      int
		User      string
		Password  string
		Name      string
		Schema    string
		Heartbeat int
		PoolSize  int
	}
	Redis struct {
		Enabled  bool
		Host     string
		Port     int
		Password string
		PoolSize int
	}
	Port           int
	Verbose        bool
	HostMountPoint string
	JwtSecret      string
}

var Conf Configuration

type RuntimeStruct struct {
	StartTime    time.Time
	InstanceName string
	Database     string
	Local        bool
	Cache        string
}

var Runtime RuntimeStruct

func Init() {

	// Load .env (as environment variables)
	if err := godotenv.Load(); err != nil {
		//fmt.Println(".env not found, continue anyway...")
	}

	Conf.Package.BuildTime = buildTime

	// Configura Viper per leggere da ENV
	viper.AutomaticEnv()

	Conf.Package.Name = name
	Conf.Package.Version = version

	Conf.DB.Enabled = viper.GetBool("DB_ENABLED")
	Conf.DB.Host = viper.GetString("DB_HOST")
	Conf.DB.Port = viper.GetInt("DB_PORT")
	Conf.DB.User = viper.GetString("DB_USER")
	Conf.DB.Password = viper.GetString("DB_PASSWORD")
	Conf.DB.Name = viper.GetString("DB_NAME")
	Conf.DB.Schema = viper.GetString("DB_SCHEMA")
	Conf.DB.Heartbeat = viper.GetInt("DB_HEARTBEAT")
	Conf.DB.PoolSize = viper.GetInt("DB_POOLSIZE")

	Conf.Redis.Enabled = viper.GetBool("REDIS_ENABLED")
	Conf.Redis.Host = viper.GetString("REDIS_HOST")
	Conf.Redis.Port = viper.GetInt("REDIS_PORT")
	Conf.Redis.Password = viper.GetString("REDIS_PASSWORD")
	Conf.Redis.PoolSize = viper.GetInt("REDIS_POOLSIZE")

	Conf.Port = viper.GetInt("PORT")
	Conf.Verbose = viper.GetBool("VERBOSE")
	Conf.HostMountPoint = viper.GetString("HOST_MOUNT_POINT")
	Conf.JwtSecret = viper.GetString("JWT_SECRET")

	Runtime.InstanceName = viper.GetString("INSTANCE_NAME")
	Runtime.StartTime = time.Now().UTC()
	Runtime.Database = "None"
	Runtime.Cache = "None"

	hostInfo, _ := host.Info()

	if Runtime.InstanceName == "" {
		Runtime.InstanceName = "EKHOES-" + hostInfo.Hostname
	}

	if err := viper.Unmarshal(&Conf); err != nil {
		log.Fatalf("errore unmarshalling config: %v", err)
	}

	//fmt.Printf("%+v\n", conf)
}

func Name() string {
	return Conf.Package.Name
}

func Version() string {
	return Conf.Package.Version
}

func BuildTime() string {
	return buildTime
}

func Local() bool {
	return Runtime.Local
}

func PosgresEnabled() bool {
	return Conf.DB.Enabled
}

func RedisEnabled() bool {
	return Conf.Redis.Enabled
}

func Port() int {
	return Conf.Port
}

func SetPort(port int) {
	Conf.Port = port
}
