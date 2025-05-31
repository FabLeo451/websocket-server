package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type User struct {
	Id                      string `json:"id" bson:"Id"`
	Name                    string `json:"name" bson:"Name"`
	Password                string `json:"password" bson:"Password"`
	Email                   string `json:"email" bson:"Email"`
	Account_Non_Expired     bool   `json:"account_Non_Expired" bson:"Account_Non_Expired"`
	Account_Non_Locked      bool   `json:"account_Non_Locked" bson:"Account_Non_Locked"`
	Credentials_Non_Expired bool   `json:"credentials_Non_Expired" bson:"Credentials_Non_Expired"`
	Enabled                 bool   `json:"enabled" bson:"Enabled"`
	Created                 string `json:"created" bson:"Created"`
	Updated                 string `json:"updates" bson:"Updated"`
}

type Host struct {
	Mem  *mem.VirtualMemoryStat
	Disk *disk.UsageStat
}

func getMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	metrics := map[string]interface{}{
		"activeConnections": atomic.LoadInt32(&activeConnections),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

func getSystemMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	var metrics Host

	// Get disk usage

	diskUsage, err := disk.Usage(conf.HostMountPoint)

	if err != nil {
		LogWrite("%s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	metrics.Disk = diskUsage

	// Get memory usage

	v, err := mem.VirtualMemory()
	if err != nil {
		panic(err)
	}

	metrics.Mem = v

	response, _ := json.Marshal(metrics)

	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func getRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: %s %s %s\n", r.RemoteAddr, r.UserAgent(), r.Method, r.URL)
	//io.WriteString(w, "This is my website!\n")

	response, _ := json.Marshal(conf.Package)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

/*
// curl -v -X POST -H "Content-Type: application/json" -d '{"name":"Fabio", "email":"fabio@leone.net", "password":"fabio"}' localhost:8080/user
func createUser(w http.ResponseWriter, r *http.Request) {
	var user User

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var stmt string
	\/*
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	*\/

	id := uuid.New()

	pattern := `insert into api.users ("id", "name", "email", "password") values ('%s', '%s', '%s', crypt('%s', gen_salt('bf')))`

	stmt = fmt.Sprintf(pattern, id, user.Name, user.Email, user.Password \/*string(hash)*\/)
	//fmt.Println(stmt)

	db := DB_GetConnection()

	if db != nil {

		_, err := db.Exec(stmt)

		if err == nil {
			LogWrite("Created user %s %s\n", user.Name, user.Email)
			w.WriteHeader(http.StatusCreated)
		} else {
			pgErr, _ := err.(*pq.Error)

			switch pgErr.Code {
			case "23505":
				http.Error(w, "Email already used by another user", http.StatusBadRequest)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

	} else {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
	}

}

func login(w http.ResponseWriter, r *http.Request) {
	var user User

	reqDump, _ := httputil.DumpRequest(r, true)

	fmt.Printf("REQUEST:\n%s\n", string(reqDump))

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//db, err := DB_Connect()
	db := DB_GetConnection()

	//if err == nil {
	if db != nil {

		query := "SELECT ID, (PASSWORD = crypt($1, PASSWORD)) AS password_match FROM api.users WHERE EMAIL = $2 AND ENABLED = true"

		rows, err := db.Query(query, user.Password, user.Email)

		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id := ""
		var password_match bool

		for rows.Next() {
			_ = rows.Scan(&id, &password_match)

			if !password_match {
				http.Error(w, "Wrong password", http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusOK)
		}

		if id == "" {
			http.Error(w, "User not found", http.StatusUnauthorized)
		}

	} else {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
	}

}
*/
