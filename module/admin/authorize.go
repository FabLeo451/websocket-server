package admin

import (
	"database/sql"
	"errors"

	"ekhoes-server/db"
)

type AuthResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Id         string `json:"id"`
	Name       string `json:"name"`
	Roles      string `json:"roles"`
	Privileges string `json:"privileges"`
}

/**
 * Authorize - Query database for user
 */
func Authorize(email string, password string) (*AuthResult, error) {

	result := &AuthResult{
		Success:    false,
		Message:    "",
		Id:         "",
		Name:       "",
		Roles:      "",
		Privileges: "",
	}

	conn := db.DB_GetConnection()

	if conn != nil {

		query, err := db.LoadSQL(SqlFS, "authorize.sql")

		if err != nil {
			return nil, err
		}

		//query = strings.ReplaceAll(query, "{{DB_SCHEMA}}", os.Getenv("DB_SCHEMA"))

		rows, err := conn.Query(query, password, email)

		if errors.Is(err, sql.ErrNoRows) {
			result.Message = "User not found"
			return result, nil
		} else if err != nil {
			return nil, err
		}

		password_match := false

		for rows.Next() {
			_ = rows.Scan(&result.Id, &result.Name, &password_match, &result.Roles, &result.Privileges)

			if !password_match {
				return nil, errors.New("Wrong password")
			}

			result.Success = true
		}

		if result.Id == "" {
			result.Message = "User not found"
			return result, nil
		}

	} else {
		return nil, errors.New("Database unavailable")
	}

	return result, nil
}
