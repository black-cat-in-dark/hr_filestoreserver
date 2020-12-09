package db

import(
	mydb "hr_filestoreserver/db/mysql"
	"fmt"
)

// 用username从表中获取token
func GetUserToken(username string)(string,error){
	var token string
	stmt,err:=mydb.DBConn().Prepare("select user_token from tbl_user_token where user_name=?")
	if err!=nil{
		fmt.Println(err.Error())
		return token,err
	}
	defer stmt.Close()

	err=stmt.QueryRow(username).Scan(&token)
	if err!=nil{
		fmt.Println(err.Error())
		return token,err
	}
	return token,nil
}

// 删除无效的token
func DeleteToken(token string) bool{
	stmt,err:=mydb.DBConn().Prepare("delete from tbl_user_token where user_token=?")
	if err!=nil{
		fmt.Println(err.Error())
		return false
	}
	ret,err:=stmt.Exec("token")
	if err!=nil{
		fmt.Println(err.Error())
		return false
	}
	if rf,_:=ret.RowsAffected();rf<=0{
		fmt.Printf("没被删除掉，可能因为token%s不存在\n",token)
	}
	return true

}