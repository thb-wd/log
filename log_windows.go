package log

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

type logWriter struct {
	orginName string
	fileName  string
	file      *os.File
	yearDay   int
}

func compress(fileName string) error {
	fw, err := os.Create(fileName + ".gz")
	if err != nil {
		return err
	}
	defer fw.Close()

	gw, err := gzip.NewWriterLevel(fw, 9)
	if err != nil {
		return err
	}
	defer gw.Close()

	fr, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer fr.Close()

	gw.Header.Name = fileName
	buf := make([]byte, 1024)
	for {
		n, err := fr.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		_, err = gw.Write(buf[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func compressAndRemove(fileName string) {
	err := compress(fileName)
	if err != nil {
		Warn(err)
		return
	}

	os.Remove(fileName)
}

func hanleOldLog(dir string) {
	rd, err := ioutil.ReadDir(dir)
	if err != nil {
		Warn(err)
		return
	}

	nowTime := time.Now()
	nowYearDay := nowTime.YearDay()
	for _, fi := range rd {
		name := dir + "\\" + fi.Name()
		if name == errorWriter.fileName || name == accessWriter.fileName {
			continue
		}

		fileYearDay := fi.ModTime().YearDay()
		offsetYearDay := nowYearDay - fileYearDay
		if offsetYearDay > 0 {
			if offsetYearDay > 7 {
				Debug("remove", name)
				err = os.Remove(name)
				if err != nil {
					Warn(err)
				}
			} else if offsetYearDay > 3 {
				if strings.Contains(name, ".gz") {
					continue
				}

				Debug("compressAndRemove", name)
				compressAndRemove(name)
			}
		}
	}
}

func (l logWriter) Write(p []byte) (n int, err error) {
	nowTime := time.Now()
	newYearDay := nowTime.YearDay()

	if l.file == nil {
		l.yearDay = newYearDay
		l.fileName = l.orginName + "." + nowTime.Format("2006-01-02") + ".log"
		l.file, _ = os.OpenFile(l.fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0766)
	} else if l.yearDay != newYearDay {
		l.file.Close()

		l.yearDay = newYearDay
		l.fileName = l.orginName + "." + nowTime.Format("2006-01-02") + ".log"
		l.file, _ = os.OpenFile(l.fileName, os.O_CREATE|os.O_WRONLY, 0766)
	}

	n, err = l.file.Write(p)
	return
}

var accessLogger *log.Logger
var errorLogger *log.Logger

var accessWriter = logWriter{}
var errorWriter = logWriter{}

var appName string

func init() {
	var path string
	path, appName = getExecDir()
	mkDir(path + "log")
	dir := path + "log/" + appName
	mkDir(dir)
	var orginName = dir + "\\" + appName

	accessWriter.orginName = orginName + "-access"
	errorWriter.orginName = orginName + "-error"
	accessLogger = log.New(accessWriter, "", log.Ldate|log.Ltime)
	errorLogger = log.New(errorWriter, "", log.Ldate|log.Ltime)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				Error("recover", err)
			}
		}()

		hanleOldLog(dir)

		var interval = 60 * 60 * 24 * time.Second
		t := time.NewTimer(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				{
					hanleOldLog(dir)
					t.Reset(interval)
				}
			}
		}
	}()
}

// Debug 调试
func Debug(v ...interface{}) {
	s := "DEBUG"
	//_, path, line, _ := runtime.Caller(1)
	//idx := strings.LastIndex(path, "/")
	//s += fmt.Sprintf(" %s %s:%d -", appName, path[idx+1:], line)
	for _, k := range v {
		s += " " + fmt.Sprint(k)
	}

	fmt.Println(s)
	accessLogger.Println(s)
}

// Info 信息
func Info(v ...interface{}) {
	s := "INFO"
	//_, path, line, _ := runtime.Caller(1)
	//idx := strings.LastIndex(path, "/")
	//s += fmt.Sprintf(" %s %s:%d -", appName, path[idx+1:], line)
	for _, k := range v {
		s += " " + fmt.Sprint(k)
	}

	fmt.Println(s)
	accessLogger.Println(s)
}

// Warn 警告错误
func Warn(v ...interface{}) {
	s := "WARN"
	//_, path, line, _ := runtime.Caller(1)
	//idx := strings.LastIndex(path, "/")
	//s += fmt.Sprintf(" %s %s:%d -", appName, path[idx+1:], line)
	for _, k := range v {
		s += " " + fmt.Sprint(k)
	}

	fmt.Println(s)
	accessLogger.Println(s)
}

// System 重要系统信息
func System(v ...interface{}) {
	s := "SYSTEM"
	//_, path, line, _ := runtime.Caller(1)
	//idx := strings.LastIndex(path, "/")
	//s += fmt.Sprintf(" %s %s:%d -", appName, path[idx+1:], line)
	for _, k := range v {
		s += " " + fmt.Sprint(k)
	}
	fmt.Println(s)
	accessLogger.Println(s)
	errorLogger.Println(s)
}

// Error 严重错误
func Error(v ...interface{}) {
	s := "ERROR"
	_, path, line, _ := runtime.Caller(1)
	idx := strings.LastIndex(path, "/")
	s += fmt.Sprintf(" %s %s:%d -", appName, path[idx+1:], line)
	for _, k := range v {
		s += " " + fmt.Sprint(k)
	}
	fmt.Println(s + string(debug.Stack()))
	accessLogger.Println(s)
	errorLogger.Println(s + string(debug.Stack()))
}

// Fatal 致命错误
func Fatal(v ...interface{}) {
	s := "FATAL"
	_, path, line, _ := runtime.Caller(1)
	idx := strings.LastIndex(path, "/")
	s += fmt.Sprintf(" %s %s:%d -", appName, path[idx+1:], line)
	for _, k := range v {
		s += " " + fmt.Sprint(k)
	}
	fmt.Println(s + string(debug.Stack()))
	accessLogger.Println(s)
	errorLogger.Println(s + string(debug.Stack()))
	os.Exit(1)
}

func mkDir(name string) {
	_, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(name, 0775)
		}
	}
}

func getExecDir() (string, string) {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	idx := strings.LastIndex(path, "\\")
	return string(path[0 : idx+1]), path[idx+1:]
}
