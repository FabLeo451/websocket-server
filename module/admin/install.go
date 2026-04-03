package admin

import (
	"ekhoes-server/db"
	"ekhoes-server/utils"
	"errors"
)

func Install() error {
	utils.Log(thisModule, "Opening database...")

	if err := db.OpenDatabase(); err != nil {
		return err
	}

	utils.Log(thisModule, "Creating schema...")

	err := db.ExecuteSQL(SqlFS, "install.sql")

	if err != nil {
		return err
	}

	db.CloseDatabase()

	return nil
}

func CreateAdmin(args ...interface{}) error {

	if err := db.OpenDatabase(); err != nil {
		return err
	}

	var email string

	if len(args) > 0 {
		if v, ok := args[0].(string); ok {
			email = v
		} else {
			return errors.New("CreateAdmin(): first argument must be a string")
		}
	}

	utils.Log(thisModule, "Creating admin user %s...", email)

	if err := db.ExecuteSQL(SqlFS, "create_user.sql", "1000", "Administrator", email, "admin", "enabled"); err != nil {
		return err
	}

	if err := db.ExecuteSQL(SqlFS, "add_role.sql", "1000", "ADMIN"); err != nil {
		return err
	}

	db.CloseDatabase()

	return nil
}
