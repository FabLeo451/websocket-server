package admin

import (
	"database/sql"
	"errors"

	"ekhoes-server/auth"
	"ekhoes-server/db"
)

type AuthResult struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	User    auth.User `json:"user"`
}

/**
 * Authenticate - Query database for user
 */
func Authenticate(email string, password string) (*AuthResult, error) {

	result := &AuthResult{}

	conn := db.DB_GetConnection()

	if conn != nil {

		query, err := db.LoadSQL(SqlFS, "authenticate.sql")

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
			_ = rows.Scan(&result.User.Id, &result.User.Name, &password_match, &result.User.Roles, &result.User.Privileges)

			if !password_match {
				result.Message = "Wrong password"
				return result, nil
			}

			result.Success = true
		}

		if result.User.Id == "" {
			result.Message = "User not found"
			return result, nil
		}

	} else {
		return nil, errors.New("Database unavailable")
	}

	result.User.Email = email

	return result, nil
}
