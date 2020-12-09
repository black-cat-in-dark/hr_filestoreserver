package mysql

import(
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	"os"
	"log"
)

var db *sql.DB

func init(){
	db,_=sql.Open("mysql","root:123456@tcp(127.0.0.1:33060)/fileserver?charset=utf8")
	db.SetMaxOpenConns(1000)
	err:=db.Ping()
	if err!=nil{
		fmt.Println("Failed to connect to mysql,err:"+err.Error())
		os.Exit(1)
	}
	
}

// DBConn : 返回数据库连接对象
func DBConn() *sql.DB {
	return db
}

func ParseRows(rows *sql.Rows) []map[string]interface{} {
	columns, _ := rows.Columns()
	scanArgs := make([]interface{}, len(columns))
	values := make([]interface{}, len(columns))
	for j := range values {
		scanArgs[j] = &values[j]
	}

	record := make(map[string]interface{})//hr 保存一行
	records := make([]map[string]interface{}, 0)
	for rows.Next() {//hr next就是一行一行的读
		//将行数据保存到record字典
		err := rows.Scan(scanArgs...)//hr Scan将next得到的row的各个值赋值到后面的指针中
		checkErr(err)

		for i, col := range values {
			if col != nil {
				record[columns[i]] = col
			}
		}
		records = append(records, record)
	}
	return records
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
}