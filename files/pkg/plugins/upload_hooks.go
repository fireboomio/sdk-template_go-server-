package plugins

import (
	"custom-go/pkg/types"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
)

type UploadBody[M any] struct {
	File  types.HookFile `json:"file"`
	Meta  M              `json:"meta"`
	Error struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"error"`
}

func RegisterUploadHook[M any](provider, profile string, hook types.UploadHook, hookFunc func(*types.UploadHookRequest, *UploadBody[M]) (*types.UploadHookResponse, error)) {
	types.AddEchoRouterFunc(func(e *echo.Echo) {
		apiPath := fmt.Sprintf("/upload/%s/%s/%s", provider, profile, hook)
		e.Logger.Debugf(`Registered uploadHook [%s]`, apiPath)
		e.POST(apiPath, buildUploadHook(hookFunc))
	})
}

func buildUploadHook[M any](hookFunc func(request *types.UploadHookRequest, body *UploadBody[M]) (*types.UploadHookResponse, error)) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		pur := c.(*types.UploadHookRequest)
		var (
			input    UploadBody[M]
			response types.UploadHookResponse
		)
		if err = c.Bind(&input); err != nil {
			response.Error = err.Error()
			return c.JSON(http.StatusInternalServerError, response)
		}

		output, err := hookFunc(pur, &input)
		if err != nil {
			response.Error = err.Error()
			return c.JSON(http.StatusInternalServerError, response)
		}
		response = *output
		return c.JSON(http.StatusOK, response)
	}
}
