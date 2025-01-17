package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
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
