package db

import (
	"database/sql"
	"zxPayPal/beelog"
	"zxPayPal/config"

	_ "github.com/go-sql-driver/mysql"
)

var MysqlDB *sql.DB

func InitSql() {
	var err error
	MysqlDB, err = sql.Open("mysql", config.Config.SqlAddr)
	if err != nil {
		beelog.Log.Error("connect database err:%s", err.Error())
	}

	err = MysqlDB.Ping()
	beelog.Log.Debug("MysqlDB:%+v", MysqlDB)
	if err != nil {
		beelog.Log.Error("mysql ping failed:%s", err.Error())
	}
}

func Query(query string, args ...interface{}) (*sql.Rows, error) {
	beelog.Log.Debug("{%s} %+v", query, args)
	return MysqlDB.Query(query, args...)
}

func Exec(query string, args ...interface{}) (sql.Result, error) {
	beelog.Log.Debug("{%s} %+v", query, args)
	return MysqlDB.Exec(query, args...)
}

func QueryRow(query string, args ...interface{}) *sql.Row {
	beelog.Log.Debug("{%s} %+v", query, args)
	return MysqlDB.QueryRow(query, args...)
}
