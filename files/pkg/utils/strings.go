package utils

import (
	"bytes"
	"encoding/json"
	"github.com/tidwall/gjson"
	"golang.org/x/exp/slices"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var validateLengthKinds = []reflect.Kind{reflect.Map, reflect.Slice}

// IsZeroValue 判断值是否为零值
func IsZeroValue(val any) bool {
	if nil == val {
		return true
	}
	if _, ok := val.(reflect.Kind); ok {
		return true
	}
	value := reflect.ValueOf(val)
	if value.IsZero() {
		return true
	}

	if slices.Contains(validateLengthKinds, value.Kind()) {
		return value.Len() == 0
	}

	return false
}

func GetStringValueWithDefault(val, defaultV string) string {
	if val == "" {
		return defaultV
	}
	return val
}

func JoinString(sep string, str ...string) string {
	if len(str) == 0 {
		return ""
	}
	return strings.Join(str, sep)
}

func JoinPathAndToSlash(path ...string) string {
	return filepath.ToSlash(filepath.Join(path...))
}

var placeholderRegexp = regexp.MustCompile(`\${([^}]+)}`)

func ReplacePlaceholder(jsonStr, str string) string {
	return placeholderRegexp.ReplaceAllStringFunc(str, func(s string) string {
		if getValue := gjson.Get(jsonStr, s[2:len(s)-1]); getValue.Exists() {
			return getValue.String()
		}

		return s
	})
}

func MarshalWithoutEscapeHTML(obj any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(buffer)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(obj)
	return buffer.Bytes(), err
}
