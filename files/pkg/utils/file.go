package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type fileCache struct {
	info    os.FileInfo
	content []byte
}

var fileCacheMap = &sync.Map{}

func ReadBytesAndCacheFile(path string) (content []byte, err error) {
	fileInfo, err := os.Stat(path)
	if nil != err {
		return
	}

	value, ok := fileCacheMap.Load(path)
	if ok {
		cache := value.(*fileCache)
		if reflect.DeepEqual(cache.info.ModTime(), fileInfo.ModTime()) {
			content = cache.content
			return
		}

		if content, err = os.ReadFile(path); err != nil {
			return
		}

		cache.info = fileInfo
		cache.content = content
		return
	}

	if content, err = os.ReadFile(path); err != nil {
		return
	}

	fileCacheMap.Store(path, &fileCache{fileInfo, content})
	return
}

func ReadStructAndCacheFile(path string, result interface{}) error {
	bytesData, err := ReadBytesAndCacheFile(path)
	if nil != err {
		return err
	}

	return json.Unmarshal(bytesData, &result)
}

var zeroTimeStr = `":"` + time.Time{}.Format(time.RFC3339) + `"`

func ClearZeroTime(data []byte) []byte {
	dataStr := string(data)
	for {
		index := strings.Index(dataStr, zeroTimeStr)
		if index == -1 {
			break
		}
		startIndex := strings.LastIndex(dataStr[:index], `"`)
		endIndex := index + len(zeroTimeStr)
		if dataStr[startIndex-1] == ',' {
			startIndex--
		}
		dataStr = dataStr[:startIndex] + dataStr[endIndex:]
	}
	return []byte(dataStr)
}

func ConvertType[S, T any](s *S) (t *T) {
	convertBytes, _ := json.Marshal(s)
	_ = json.Unmarshal(ClearZeroTime(convertBytes), &t)
	return
}

func GetCallerName(prefix string) string {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	_, callerFilename, _, _ := runtime.Caller(2)
	_, callerName, ok := strings.Cut(callerFilename, prefix)
	if !ok {
		return ""
	}

	return strings.TrimSuffix(callerName, filepath.Ext(callerName))
}

func NotExistFile(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}
