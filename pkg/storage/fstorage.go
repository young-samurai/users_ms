package fstorage

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
)

// Storage - storage type
type Storage struct {
	conn     *pgx.Conn
	settings *StorageParams
	tx       pgx.Tx
}

// QueryParams - query parameters
// Dest - destination variable
// QueryString - query string
// Args - query arguments
type QueryParams struct {
	Dest        interface{}
	QueryString string
	Args        []interface{}
}

// CopyParams - copy parameters
// TabName - table name
// Fields - slice of field names
// Source - data source in format [][]interface{}
type CopyParams struct {
	TabName string
	Fields  []string
	Source  interface{}
}

// StorageParams - storage parameters
// Ctx - context
// Dsn - database name
// AppName - name of application
// AutoClose - automatically close connection
// AutoCommit - automatically commit transaction
type StorageParams struct {
	Ctx        context.Context
	Dsn        string
	AppName    string
	AutoClose  bool
	AutoCommit bool
}

// NewStorage - constructor function
func NewStorage(storageParams *StorageParams) (*Storage, error) {

	// Checking settings
	if storageParams == nil {
		return nil, errors.New("not enough storage settings")
	}

	// Vars
	var (
		connConfig *pgx.ConnConfig
		conn       *pgx.Conn
		err        error
	)

	// Create config
	connConfig, err = makeConnConfig(storageParams)
	if err != nil {
		return nil, err
	}

	// Open connection uses config
	conn, err = pgx.ConnectConfig(storageParams.Ctx, connConfig)
	if err != nil {
		return nil, err
	}

	return &Storage{
		conn:     conn,
		settings: storageParams,
	}, nil
}

// Methods

// TxBegin - beginning transaction
func (s *Storage) TxBegin() (pgx.Tx, error) {
	// Beginning transaction
	tx, err := s.conn.BeginTx(s.settings.Ctx, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.NotDeferrable,
	})
	if err != nil {
		return nil, err
	}
	s.tx = tx
	return tx, nil
}

// TxEnd - ending transaction
func (s *Storage) TxEnd() error {
	return s.tx.Commit(s.settings.Ctx)
}

// TxRollback - rollback transaction
func (s *Storage) TxRollback() error {
	return s.tx.Rollback(s.settings.Ctx)
}

// TxCommit - commit transaction
func (s *Storage) TxCommit() error {
	return s.tx.Commit(s.settings.Ctx)
}

// RunQuery - run query and return result
func (s *Storage) RunQuery(queryParams *QueryParams) error {

	if s == nil || s.conn == nil {
		return errors.New("storage is not ready to query running")
	}

	// Beginning transaction
	var (
		tx  pgx.Tx
		err error
	)

	if s.tx == nil {
		tx, err = s.TxBegin()
		if err != nil {
			return err
		}
		s.tx = tx
	} else {
		tx = s.tx
	}

	err = pgxscan.Select(s.settings.Ctx, tx, queryParams.Dest, queryParams.QueryString, queryParams.Args...)
	if err != nil {
		if s.settings.AutoCommit {
			e := tx.Rollback(s.settings.Ctx)
			if e != nil {
				return e
			}
			return err
		} else {
			return err
		}
	}

	if s.settings.AutoCommit {
		e := tx.Commit(s.settings.Ctx)
		if e != nil {
			return e
		}
	}

	if s.settings.AutoClose {
		err = s.Release()
		if err != nil {
			return err
		}
	}
	return nil
}

// ExecuteQuery - execute query and return rows affected
func (s *Storage) ExecuteQuery(queryParams *QueryParams) (int64, error) {

	if s == nil || s.conn == nil {
		return 0, errors.New("storage is not ready to query running")
	}

	// Beginning transaction
	var (
		tx  pgx.Tx
		err error
	)

	if s.tx == nil {
		tx, err = s.TxBegin()
		if err != nil {
			return 0, err
		}
		s.tx = tx
	} else {
		tx = s.tx
	}

	// err = pgxscan.Select(s.settings.Ctx, tx, queryParams.Dest, queryParams.QueryString, queryParams.Args...)
	res, err := tx.Exec(s.settings.Ctx, queryParams.QueryString, queryParams.Args...)
	if err != nil || res.RowsAffected() == 0 {
		if s.settings.AutoCommit {
			e := tx.Rollback(s.settings.Ctx)
			if e != nil {
				return 0, e
			}
			return 0, err
		} else {
			return 0, err
		}
	}

	if s.settings.AutoCommit {
		e := tx.Commit(s.settings.Ctx)
		if e != nil {
			return 0, e
		}
	}

	if s.settings.AutoClose {
		err = s.Release()
		if err != nil {
			return 0, err
		}
	}

	return res.RowsAffected(), nil
}

// Release - release connection
func (s *Storage) Release() error {
	return s.conn.Close(s.settings.Ctx)
}

// CopyToDatabase - copy data from source to database
func (s *Storage) CopyToDatabase(copyParams *CopyParams) error {

	var (
		data [][]interface{}
		err  error
	)

	// Checking connection
	if s == nil || s.conn == nil {
		return errors.New("storage is not ready to copy data to database")
	}

	// Checking data
	switch copyParams.Source.(type) {
	case [][]interface{}:
		data = copyParams.Source.([][]interface{})
	default:
		data, err = convertToInterfaceSlice(copyParams.Source)
		if err != nil {
			return err
		}
	}

	// Beginning transaction
	var (
		tx pgx.Tx
	)

	if s.tx == nil {
		tx, err = s.TxBegin()
		if err != nil {
			return err
		}
		s.tx = tx
	} else {
		tx = s.tx
	}

	// Copying data
	_, err = tx.CopyFrom(
		s.settings.Ctx,
		pgx.Identifier{copyParams.TabName},
		copyParams.Fields,
		pgx.CopyFromSlice(len(data), func(i int) ([]interface{}, error) {
			return data[i], nil
		}),
	)

	// Ending transaction
	if err != nil {
		if s.settings.AutoCommit {
			e := tx.Rollback(s.settings.Ctx)
			if e != nil {
				return e
			}
			return err
		} else {
			return err
		}
	}

	if s.settings.AutoCommit {
		e := tx.Commit(s.settings.Ctx)
		if e != nil {
			return e
		}
	}
	return nil
}

// UpdateFromSlice - update data from source in database
func (s *Storage) UpdateFromSlice(qp *CopyParams) error {

	var (
		data [][]interface{}
		err  error
	)

	// Checking connection
	if s == nil || s.conn == nil {
		return errors.New("storage is not ready to copy data to database")
	}

	// Checking data
	switch qp.Source.(type) {
	case [][]interface{}:
		data = qp.Source.([][]interface{})
	default:
		data, err = convertToInterfaceSlice(qp.Source)
		if err != nil {
			return err
		}
	}

	// Beginning transaction
	var (
		tx pgx.Tx
	)

	if s.tx == nil {
		tx, err = s.TxBegin()
		if err != nil {
			return err
		}
		s.tx = tx
	} else {
		tx = s.tx
	}

	// Updating data
	for _, row := range data {
		// creating row of query to update
		query := "UPDATE " + qp.TabName + " SET "
		for i, field := range qp.Fields {
			if i != 0 {
				query += ", "
			}
			query += field + " = $" + strconv.Itoa(i+1)
		}
		query += " WHERE id = $" + strconv.Itoa(len(qp.Fields)+1)

		// adding arguments
		args := make([]interface{}, len(row)+1)
		copy(args, row)
		args[len(row)] = row[0]
		args = args[1:]

		// run query
		_, err = tx.Exec(s.settings.Ctx, query, args...)
		if err != nil {
			break
		}
	}
	// Ending transaction
	if err != nil {
		if s.settings.AutoCommit {
			e := tx.Rollback(s.settings.Ctx)
			if e != nil {
				return e
			}
			return err
		} else {
			return err
		}
	}
	if s.settings.AutoCommit {
		e := tx.Commit(s.settings.Ctx)
		if e != nil {
			return e
		}
	}
	return nil
}

// Update - update data in database from slice of interfaces
func (s *Storage) Update(qp *CopyParams) error {

	// Checking connection
	if s == nil || s.conn == nil {
		return errors.New("storage is not ready to copy data to database")
	}

	// Checking data in struct
	if qp.TabName == "" {
		return errors.New("tab name is empty")
	}
	if len(qp.Fields) == 0 {
		return errors.New("fields is empty")
	}
	if qp.Source == nil {
		return errors.New("source is empty")
	}

	// Checking count of fields in source
	sourceSlice, ok := qp.Source.([][]interface{})
	if !ok {
		return errors.New("source is not a slice")
	}
	if len(qp.Fields) != len(sourceSlice[0]) {
		return errors.New("fields count is not equal to source")
	}

	// Beginning transaction
	var (
		tx pgx.Tx
		e  error
	)

	if s.tx == nil {
		tx, e = s.TxBegin()
		if e != nil {
			return e
		}
		s.tx = tx
	} else {
		tx = s.tx
	}

	// Generating and executing queries
	for _, row := range sourceSlice {
		textQuery, args := generateUpdateQuery(qp.TabName, qp.Fields, row)
		_, e = tx.Exec(s.settings.Ctx, textQuery, args...)
		if e != nil {
			err := tx.Rollback(s.settings.Ctx)
			if err != nil {
				return err
			}
			return e
		}
	}

	// Ending transaction
	if s.settings.AutoCommit {
		e := tx.Commit(s.settings.Ctx)
		if e != nil {
			return e
		}
	}
	return nil
}

// Ping - check connection
func (s *Storage) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// Utils functions

// convertToInterfaceSlice - convert slice to interface slice
func convertToInterfaceSlice(data interface{}) ([][]interface{}, error) {
	sliceValue := reflect.ValueOf(data)
	if sliceValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("data is not a slice")
	}
	sliceLen := sliceValue.Len()
	if sliceLen == 0 {
		return nil, nil
	}
	elemValue := sliceValue.Index(0)
	if !elemValue.IsValid() {
		return nil, fmt.Errorf("slice element is not valid")
	}
	if !elemValue.Elem().CanInterface() {
		return nil, fmt.Errorf("slice element fields are not exported")
	}
	rows := make([][]interface{}, sliceLen)
	for i := 0; i < sliceLen; i++ {
		elemValue = sliceValue.Index(i)
		if !elemValue.Elem().CanInterface() {
			return nil, fmt.Errorf("slice element fields are not exported")
		}
		fieldsValue := elemValue.Elem()

		// Delete 5 systems fields
		// state, sizeCache, unknownFields, AuthToken, ID
		fieldCount := fieldsValue.NumField() - 5
		if fieldCount < 0 {
			return nil, fmt.Errorf("not enough fields in slice")
		}

		row := make([]interface{}, fieldCount)
		for j := 0; j < fieldCount; j++ {
			field := fieldsValue.Field(j)
			if !field.CanInterface() {
				// fmt.Errorf("field %s is not exported", fieldsValue.Type().Field(j).Name)
				continue
			}
			row[j] = field.Interface()
		}
		rows[i] = row
	}
	return rows, nil
}

// makeConnConfig - make connection config
func makeConnConfig(sp *StorageParams) (*pgx.ConnConfig, error) {
	// Parse config
	config, err := pgx.ParseConfig(sp.Dsn)
	if err != nil {
		return nil, err
	}
	// Adding runtime params to config
	config.RuntimeParams = map[string]string{
		"application_name":                    sp.AppName,
		"statement_timeout":                   "60s",
		"idle_in_transaction_session_timeout": "60s",
	}
	return config, nil
}

// generateUpdateQuery - generate update query
func generateUpdateQuery(tabName string, fields []string, row []interface{}) (string, []interface{}) {

	// vars
	setClauses := make([]string, 0, len(fields))
	whereClauses := make([]string, 0)
	args := make([]interface{}, 0)

	for i, field := range fields {
		if strings.HasPrefix(field, "@") {
			whereClauses = append(whereClauses, fmt.Sprintf("%s=$%d", strings.TrimPrefix(field, "@"), len(args)+1))
		} else {
			if len(setClauses) <= i {
				setClauses = append(setClauses, fmt.Sprintf("%s=$%d", field, len(args)+1))
			} else {
				setClauses[i] = fmt.Sprintf("%s=$%d", field, len(args)+1)
			}
		}
		args = append(args, row[i])
	}

	setClause := strings.Join(setClauses, ", ")
	whereClause := strings.Join(whereClauses, " AND ")

	updateQuery := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tabName, setClause, whereClause)
	return updateQuery, args
}