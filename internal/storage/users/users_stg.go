package users

import (
	"context"

	"users/internal/api"
	cfg "users/pkg/config"
	lgr "users/pkg/logger"
	fs "users/pkg/storage"
)

func CreateUser(ctx context.Context, req *api.UsersRequest) []*api.DBUser {
	lgr.LOG.Info("-->> ", "users.CreateUser()")

	var (
		users   []*api.DBUser
		storage *fs.Storage
	)

	//Generate query
	lgr.LOG.Info("_ACTION_: ", "generate query")
	q := GenQueryUsersInsert(req)
	lgr.LOG.Info("QUERY: ", q)

	//Create storage parameters
	lgr.LOG.Info("_ACTION_: ", "creating storage parameters")
	sp := &fs.StorageParams{
		Ctx:        ctx,
		Dsn:        cfg.CFG.Database.URL,
		AppName:    "UsersMs::CreateUser()",
		AutoCommit: true,
		AutoClose:  true,
	}

	//Create query parameters
	lgr.LOG.Info("_ACTION_: ", "creating query parameters")
	qp := &fs.QueryParams{
		Dest:        &users,
		QueryString: q,
	}

	//Open storage
	lgr.LOG.Info("_ACTION_: ", "opening connection")
	storage, err := fs.NewStorage(sp)
	if err != nil {
		lgr.LOG.Error("_ERR_: ", err)
		return nil
	}

	//Execute query
	lgr.LOG.Info("_ACTION_: ", "running query")
	err = storage.RunQuery(qp)
	if err != nil {
		lgr.LOG.Error("_ERR_: ", err)
		return nil
	}

	//Check result
	if len(users) == 0 {
		lgr.LOG.Warn("_WARN_: ", "query doesn`t return any strings")
		return nil
	}

	lgr.LOG.Info("<<-- ", "users.CreateUser()")
	return users
}
