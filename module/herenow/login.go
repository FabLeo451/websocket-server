package herenow

import (
	"ekhoes-server/auth"
	"ekhoes-server/db"
	"ekhoes-server/utils"
	"encoding/json"
	"errors"
	"net/http"
)

/**
 * POST /login
 * -H "x-user-agent: Radar/1.0.0" -H "x-platform: Android" -d '{ email: "admin@hal9k.net", password: "admin" }'
 */
func Login(w http.ResponseWriter, r *http.Request) {
	var (
		credentials auth.Credentials
		//user        auth.User
	)

	err := json.NewDecoder(r.Body).Decode(&credentials)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn := db.DB_GetConnection()

	if conn != nil {

	} else {
		utils.LogErr(thisModule, errors.New("Database unavailable"))
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//w.Write([]byte(fmt.Sprintf(`{"token":"%s", "name":"%s", "id":"%s", "hostname":"%s" }`, token, user.Name, user.Id, hostname)))

	//fmt.Println(token)

	utils.Log(thisModule, "%s successfully authenticated\n", credentials.Email)
}
