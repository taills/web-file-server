package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

var listenPort uint
var workDir string
var isHash bool
var username, password string

func main() {
	flag.UintVar(&listenPort, "port", 9091, "Listen port")
	flag.StringVar(&workDir, "dir", "./work", "Working directory")
	flag.BoolVar(&isHash, "hash", false, "Hash file name")
	flag.StringVar(&username, "username", "admin", "BasicAuth username")
	flag.StringVar(&password, "password", "admin", "BasicAuth password")
	flag.Parse()

	// 启动web服务并接收post请求，实现文件上传
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// 设置工作目录
	os.MkdirAll(workDir, 0777)
	// 切换工作目录
	if err = os.Chdir(workDir); err != nil {
		panic(err)
		return
	}
	handle := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			file, handler, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Error Retrieving the File", http.StatusBadRequest)
				return
			}
			defer file.Close()
			var newFilename string
			if isHash {
				hash := md5.New()
				hash.Write([]byte(fmt.Sprintf("%s:%s:%d", time.Now().String(), handler.Filename, handler.Size)))
				newFilename = fmt.Sprintf("%x", hash.Sum(nil))
			} else {
				newFilename = path.Base(handler.Filename)
			}

			if len(newFilename) < 3 {
				http.Error(w, "Error File Name", http.StatusBadRequest)
				return
			}
			saveFilename := filenameSafe(filepath.Join(r.URL.Path, newFilename))
			os.MkdirAll(filepath.Dir(saveFilename), 0755)
			f, err := os.OpenFile(saveFilename, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				http.Error(w, "Error Saving the File", http.StatusBadRequest)
				return
			}
			defer f.Close()
			_, err = io.Copy(f, file)
			if err != nil {
				http.Error(w, "Error Saving the File", http.StatusBadRequest)
				return
			}
			logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			_, err = logFile.WriteString(
				fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s\n",
					time.Now().Format("2006-01-02 15:04:05"),
					r.RemoteAddr,
					r.Header.Get("CF-Connecting-IP"),
					handler.Size,
					handler.Filename,
					saveFilename,
					r.UserAgent()))
			if err != nil {
				http.Error(w, "Error Saving the Log", http.StatusBadRequest)
				return
			}
			defer logFile.Close()
			w.Write([]byte("OK\n"))
			w.WriteHeader(http.StatusOK)
			return
		}
		if http.MethodGet == r.Method {
			// 要求进行 BasicAuth 认证
			u, p, ok := r.BasicAuth()
			if !ok || username != u || password != p {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized.", http.StatusUnauthorized)
				return
			}

			filename := filenameSafe(r.URL.Path)
			// 判断是否是目录
			fileInfo, err := os.Stat(filename)
			if err != nil {
				http.Error(w, "Error Reading the File", http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("delete") == "1" {
				// 判断是否是删除请求
				err := os.RemoveAll(filename)
				if err != nil {
					http.Error(w, "Error Deleting the File", http.StatusBadRequest)
					return
				}
				// 跳转到根目录
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			if fileInfo.IsDir() {
				files, err := os.ReadDir(filename)
				if err != nil {
					http.Error(w, "Error Reading the File", http.StatusBadRequest)
					return
				}
				// 输出http头 utf-8
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				for _, file := range files {
					// 写出文件名，并包含超链接
					w.Write([]byte(fmt.Sprintf("<p><a href=\"%s\">%s</a> \n", filepath.Join(r.URL.Path, file.Name()), file.Name())))
					// 包含删除超链接
					w.Write([]byte(fmt.Sprintf(" <a href=\"%s?delete=1\">Delete</a></p>\n", filepath.Join(r.URL.Path, file.Name()))))
				}
				w.WriteHeader(http.StatusOK)
				return
			} else {
				// 否则下载文件
				file, err := os.Open(filenameSafe(r.URL.Path))
				if err != nil {
					http.Error(w, "Error Reading the File", http.StatusBadRequest)
					return
				}
				defer file.Close()
				// 输出文件下载http头
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", path.Base(r.URL.Path)))
				w.Header().Set("Content-Type", "application/octet-stream")
				io.Copy(w, file)
				return
			}
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})
	// 启动http服务
	err = http.Serve(listener, handle)
	if err != nil {
		panic(err)
	}
}

// filenameSafe 过滤文件名中的非法字符
func filenameSafe(filename string) string {
	abs, err := filepath.Abs(filepath.Join(workDir, filename))
	if err != nil {
		return ""
	}
	return abs
}
