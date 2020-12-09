package handler

import(
	"net/http"
	"io/ioutil"
	"io"
	"fmt"
	"os"
	"time"
	"hr_filestoreserver/meta"
	"hr_filestoreserver/util"
	"hr_filestoreserver/store/oss"
	cmn "hr_filestoreserver/common"
	cfg "hr_filestoreserver/config"
	"hr_filestoreserver/mq"
	dblayer "hr_filestoreserver/db"
	"encoding/json"
	"strconv"
)

func UploadHandler(w http.ResponseWriter,r *http.Request){
	if r.Method=="GET"{
		//返回上传html页面
		data,err:=ioutil.ReadFile("./static/view/index.html")
		if err!=nil{
			io.WriteString(w,"internal server err"+err.Error())
			return
		}
		io.WriteString(w, string(data))
	}else if r.Method=="POST"{
		//接收文件流及存储到本地目录
		file,head,err:=r.FormFile("file")//返回句柄，头，错误
		if err!=nil{
			fmt.Printf("Failed to get data,err:%s\n", err.Error())
			return
		}
		defer file.Close()
		
		curPath,_:=os.Getwd()
		fileMeta:=meta.FileMeta{
			FileName:head.Filename,
			Location:curPath+"/tmp/"+head.Filename,
			UploadAt:time.Now().Format("2006-01-02 15:04:05"),
		}
		fmt.Println(fileMeta.Location)
		newFile,err:=os.Create(fileMeta.Location)
		if err!=nil{
			fmt.Printf("Failed to create file,err:%s\n", err.Error())
			return
		}
		defer newFile.Close()
		fileMeta.FileSize,err=io.Copy(newFile,file)
		if err!=nil{
			fmt.Printf("Failed to save data into file,err:%s\n", err.Error())
			return
		}
		newFile.Seek(0,0)
		fileMeta.FileSha1=util.FileSha1(newFile)
		fmt.Println(fileMeta.FileName+"的sha1 hash是 "+fileMeta.FileSha1)

		//hr 加入oss
		newFile.Seek(0,0)
		ossPath:="oss/"+fileMeta.FileSha1
		// err=oss.Bucket().PutObject(ossPath,newFile) //hr 这里的第二个参数要ioreader,但给了file 也没报错，说明二者同源
		// if err!=nil{
		// 	fmt.Println(err.Error())
		// 	w.Write([]byte("upload to oss failed"))
		// 	return
		// }
		// fileMeta.Location=ossPath
		data := mq.TransferData{
			FileHash:      fileMeta.FileSha1,
			CurLocation:   fileMeta.Location,
			DestLocation:  ossPath,
			DestStoreType: cmn.StoreOSS,
		}
		pubData, _ := json.Marshal(data)
		pubSuc := mq.Publish(
			cfg.TransExchangeName,
			cfg.TransOSSRoutingKey,
			pubData,
		)
		if !pubSuc {
			// TODO: 当前发送转移信息失败，稍后重试
		}

		//meta.UpdateFileMeta(fileMeta)
		_=meta.UpdateFileMetaDB(fileMeta)

		//更新用户文件表
		r.ParseForm()
		username:=r.Form.Get("username")
		suc:=dblayer.OnUserFileUploadFinished(username, fileMeta.FileSha1,
			 fileMeta.FileName, fileMeta.FileSize)
		if suc{
			http.Redirect(w, r,"/file/upload/suc", http.StatusFound)
		}else{
			w.Write([]byte("hr Update UserFile Failed"))
		}
	}
}

//上传已完成
func UploadSucHandler(w http.ResponseWriter,r *http.Request){
	io.WriteString(w,"<p>Upload finished</p> hr")
}

//获取文件元信息
func GetFileMetaHandler(w http.ResponseWriter,r *http.Request){
	r.ParseForm()

	filehash:=r.Form["filehash"][0]
	//fmeta:=meta.GetFileMeta(filehash)
	fmeta,err:=meta.GetFileMetaDB(filehash)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data,err:=json.Marshal(fmeta)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// FileQueryHandler : 查询批量的文件元信息
func FileQueryHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	limitCnt, _ := strconv.Atoi(r.Form.Get("limit"))
	username := r.Form.Get("username")
	//fileMetas, _ := meta.GetLastFileMetasDB(limitCnt)
	userFiles, err := dblayer.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(userFiles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// DownloadHandler : 文件下载接口
func DownloadHandler(w http.ResponseWriter,r *http.Request){
	r.ParseForm()
	fsha1:=r.Form.Get("filehash")
	fm:=meta.GetFileMeta(fsha1)
	
	f,err:=os.Open(fm.Location)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data,err:=ioutil.ReadAll(f)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type","application/octect-stream")
	w.Header().Set("Content-Description","attachment; filename=\""+fm.FileName+"\"")
	w.Write(data)

}

//更新元信息接口（重命名）
func FileMetaUpdateHandler(w http.ResponseWriter,r *http.Request){
	r.ParseForm()

	opType:=r.Form.Get("op")
	fileSha1:=r.Form.Get("filehash")
	newFileName:=r.Form.Get("filename")

	if opType!="0"{//作为允许更新的判断
		w.WriteHeader(http.StatusForbidden)//403禁止
		return
	}
	if r.Method!="POST"{
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}


	curFileMeta:=meta.GetFileMeta(fileSha1)
	curFileMeta.FileName=newFileName
	meta.UpdateFileMeta(curFileMeta)

	data,err:=json.Marshal(curFileMeta)
	if err!=nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// FileDeleteHandler : 删除文件及元信息
func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fileSha1 := r.Form.Get("filehash")

	fMeta := meta.GetFileMeta(fileSha1)
	// 删除文件
	os.Remove(fMeta.Location)
	// 删除文件元信息
	meta.RemoveFileMeta(fileSha1)
	// TODO: 删除表文件信息

	w.WriteHeader(http.StatusOK)
}


// TryFastUploadHandler : 尝试秒传接口
func TryFastUploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// 1. 解析请求参数
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filename := r.Form.Get("filename")
	filesize, _ := strconv.Atoi(r.Form.Get("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	fileMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 3. 查不到记录则返回秒传失败
	if fileMeta == nil {
		resp := util.RespMsg{
			Code: -1,
			Msg:  "秒传失败，请访问普通上传接口",
		}
		w.Write(resp.JSONBytes())
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	suc := dblayer.OnUserFileUploadFinished(
		username, filehash, filename, int64(filesize))
	if suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "秒传成功",
		}
		w.Write(resp.JSONBytes())
		return
	}
	resp := util.RespMsg{
		Code: -2,
		Msg:  "秒传失败，请稍后重试",
	}
	w.Write(resp.JSONBytes())
	return
}

// DownloadURLHandler : 生成文件的下载地址
func DownloadURLHandler(w http.ResponseWriter,r *http.Request){
	r.ParseForm()
	filehash:=r.Form.Get("filehash")
	// 从文件表查找记录
	row,_:=dblayer.GetFileMeta(filehash)
	// TODO: 判断文件存在OSS，还是Ceph，还是在本地

	signedURL:=oss.DownloadURL(row.FileAddr.String)
	w.Write([]byte(signedURL))

}