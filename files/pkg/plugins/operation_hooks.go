package plugins

import (
	"custom-go/pkg/types"
	"custom-go/pkg/utils"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/labstack/echo/v4"
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
		_ = jsonparser.ObjectEach(input, func(key []byte, value []byte, dataType jsonparser.ValueType, _ int) error {
			switch dataType {
			case jsonparser.Boolean, jsonparser.Null:
				output, _ = jsonparser.Set(output, value, "input", string(key))
			}
			return nil
		}, "input")
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
		c.Response().WriteHeader(http.StatusOK)

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

		outBytes, err := json.Marshal(out)
		if err != nil {
			return err
		}
		if rewriteFunc, ok := resolveRewriteFuncs[hook]; ok {
			outBytes = rewriteFunc(bodyBytes, outBytes)
		}
		return c.JSONBlob(http.StatusOK, outBytes)
	}
}
