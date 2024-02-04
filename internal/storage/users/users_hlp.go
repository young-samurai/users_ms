package users

import (
	"fmt"

	"users/internal/api"
	lgr "users/pkg/logger"
)

func GenQueryUsersInsert(req *api.UsersRequest) string {
	lgr.LOG.Info("-->> ", "users.GenQueryUsersInsert()")

	query := fmt.Sprintf(`with inserted_user as(insert into users_ref (
		login, 
		password
	) values (
		'%s', '%s'
	) returning *) select `+users_query_string+`from inserted_user as usr`, req.Login, req.Password)

	lgr.LOG.Info("<<-- ", "users.GenQueryUsersInsert()")
	return query
}

// возвращаемые от базы поля
const users_query_string = `usr.id as id, 
usr.login as login, 
usr.password as password, 
usr.is_deleted as is_deleted, 
to_char(usr.created_at, 'YYYY-MM-DD HH24:MI:SS'::text) as created_at, 
to_char(usr.updated_at, 'YYYY-MM-DD HH24:MI:SS'::text) as updated_at `
