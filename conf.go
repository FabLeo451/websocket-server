package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Package struct {
		Name    string
		Version string
	}
	DB struct {
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

var conf Config

func Init() {

	// Carica il file .env (come variabili d'ambiente)
	if err := godotenv.Load(); err != nil {
		log.Println(".env non trovato, continuo lo stesso...")
	}

	// Configura Viper per leggere da ENV
	viper.AutomaticEnv()

	conf.Package.Name = "Websocket server"
	conf.Package.Version = "1.0.0-alpha5"

	conf.DB.Host = viper.GetString("DB_HOST")
	conf.DB.Port = viper.GetInt("DB_PORT")
	conf.DB.User = viper.GetString("DB_USER")
	conf.DB.Password = viper.GetString("DB_PASSWORD")
	conf.DB.Name = viper.GetString("DB_NAME")
	conf.DB.Schema = viper.GetString("DB_SCHEMA")
	conf.DB.Heartbeat = viper.GetInt("DB_HEARTBEAT")
	conf.DB.PoolSize = viper.GetInt("DB_POOLSIZE")

	conf.Redis.Host = viper.GetString("REDIS_HOST")
	conf.Redis.Port = viper.GetInt("REDIS_PORT")
	conf.Redis.Password = viper.GetString("REDIS_PASSWORD")
	conf.Redis.PoolSize = viper.GetInt("REDIS_POOLSIZE")

	conf.Port = viper.GetInt("PORT")
	conf.Verbose = viper.GetBool("VERBOSE")
	conf.HostMountPoint = viper.GetString("HOST_MOUNT_POINT")
	conf.JwtSecret = viper.GetString("JWT_SECRET")

	if err := viper.Unmarshal(&conf); err != nil {
		log.Fatalf("errore unmarshalling config: %v", err)
	}

	//fmt.Printf("%+v\n", conf)
}
