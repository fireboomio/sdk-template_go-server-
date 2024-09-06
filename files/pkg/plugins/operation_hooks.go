package plugins

import (
	"custom-go/pkg/types"
	"custom-go/pkg/utils"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"net/http"
	"strconv"
)

func HeadersToObject(headers http.Header) types.RequestHeaders {
	obj := make(types.RequestHeaders)
	for key, values := range headers {
		if len(values) > 0 {
			obj[key] = values[0]
		}
	}
	return obj
}

func MakeDataAnyMap(data any) map[string]any {
	return map[string]any{"data": data}
}

func (m *Meta[I, O]) RegisterHook(hook types.MiddlewareHook, resolve func(*types.HookRequest, *types.OperationBody[I, O]) (*types.OperationBody[I, O], error)) {
	registerHook[I, O](m.Path, hook, resolve)
}

func (m *Subscriber[I, O]) RegisterHook(hook types.MiddlewareHook, resolve func(*types.HookRequest, *types.OperationBody[I, O]) (*types.OperationBody[I, O], error)) {
	registerHook[I, O](m.Path, hook, resolve)
}

func registerHook[I, O any](path string, hook types.MiddlewareHook, resolve func(*types.HookRequest, *types.OperationBody[I, O]) (*types.OperationBody[I, O], error)) {
	types.AddEchoRouterFunc(func(e *echo.Echo) {
		apiPath := fmt.Sprintf("/operation/%s/%s", path, hook)
		e.Logger.Debugf(`Registered operationHook [%s]`, apiPath)
		e.POST(apiPath, buildOperationHook(path, hook, resolve))
	})
}

const maximumRecursionLimit = 16

func requestContext(c echo.Context) (result *types.HookRequest, err error) {
	body := make(map[string]interface{})
	if err := c.Request().ParseForm(); err != nil {
		return result, err
	}

	result = c.(*types.HookRequest)
	for key, value := range c.Request().Form {
		body[key] = value[0]
	}
	if cycleCounter, ok := body["cycleCounter"].(int); ok {
		if cycleCounter > maximumRecursionLimit {
			return result, fmt.Errorf("maximum recursion limit reached (%d)", maximumRecursionLimit)
		}
		result.InternalClient = result.InternalClient.WithHeaders(types.RequestHeaders{"Wg-Cycle-Counter": strconv.Itoa(cycleCounter)})
	}
	return result, nil
}

var resolveRewriteFuncs = map[types.MiddlewareHook]func([]byte, []byte) []byte{
	types.MiddlewareHook_mutatingPreResolve: func(input, output []byte) []byte {
		zeros := &zeroValues{}
		inputValue, inputType, _, _ := jsonparser.Get(input, "input")
		zeros.search(inputValue, inputType, "input")
		output = zeros.set(output)
		return utils.ClearZeroTime(output)
	},
	types.MiddlewareHook_mutatingPostResolve: func(_, output []byte) []byte {
		if anyResult := gjson.GetBytes(output, "response.dataAny"); anyResult.Exists() {
			output, _ = sjson.SetRawBytes(output, "response.data", []byte(anyResult.Raw))
		}
		return output
	},
}

func buildOperationHook[I, O any](operationPath string, hook types.MiddlewareHook,
	resolve func(hook *types.HookRequest, body *types.OperationBody[I, O]) (*types.OperationBody[I, O], error)) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		var in *types.OperationBody[I, O]
		if err = json.Unmarshal(bodyBytes, &in); err != nil {
			return
		}

		hookRequest, err := requestContext(c)
		if err != nil {
			return
		}

		in.Op = operationPath
		in.Hook = hook
		in.SetClientRequestHeaders = HeadersToObject(c.Request().Header)
		out, err := resolve(hookRequest, in)
		if err != nil {
			return err
		}
		if out == nil {
			return c.JSON(http.StatusOK, in)
		}

		outBytes, err := utils.MarshalWithoutEscapeHTML(out)
		if err != nil {
			return err
		}
		if rewriteFunc, ok := resolveRewriteFuncs[hook]; ok {
			outBytes = rewriteFunc(bodyBytes, outBytes)
		}
		return c.JSONBlob(http.StatusOK, outBytes)
	}
}

type zeroValue struct {
	path      []string
	value     []byte
	valueType jsonparser.ValueType
}

type zeroValues []*zeroValue

func (v *zeroValues) set(output []byte) []byte {
	for _, item := range *v {
		_, typeInOut, _, _ := jsonparser.Get(output, item.path...)
		if typeInOut != jsonparser.NotExist {
			continue
		}
		itemValue := item.value
		if item.valueType == jsonparser.String {
			itemValue = []byte(strconv.Quote(string(item.value)))
		}
		output, _ = jsonparser.Set(output, itemValue, item.path...)
	}
	return output
}

func (v *zeroValues) add(value []byte, valueType jsonparser.ValueType, path ...string) {
	*v = append(*v, &zeroValue{path: path, value: value, valueType: valueType})
}

func (v *zeroValues) search(data []byte, dataType jsonparser.ValueType, path ...string) {
	switch dataType {
	case jsonparser.Null:
		v.add(data, dataType, path...)
	case jsonparser.String:
		if len(data) == 0 {
			v.add(data, dataType, path...)
		}
	case jsonparser.Boolean:
		if !cast.ToBool(string(data)) {
			v.add(data, dataType, path...)
		}
	case jsonparser.Number:
		if cast.ToInt(string(data)) == 0 {
			v.add(data, dataType, path...)
		}
	case jsonparser.Object:
		_ = jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, _ int) error {
			v.search(value, dataType, appendItem(path, string(key))...)
			return nil
		})
	case jsonparser.Array:
		var index int
		_, _ = jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, _ int, _ error) {
			v.search(value, dataType, appendItem(path, fmt.Sprintf("[%d]", index))...)
			index++
		})
	}
	return
}

func appendItem(array []string, item string) []string {
	result := make([]string, len(array)+1)
	copy(result, array)
	result[len(array)] = item
	return result
}
