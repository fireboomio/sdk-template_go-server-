package plugins

import (
	"custom-go/pkg/base"
	"github.com/labstack/echo/v4"
	"net/http"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type (
	httpProxyHookFunction func(*base.HttpTransportHookRequest, *HttpTransportBody) (*base.ClientResponse, error)
	httpProxyHook         struct {
		rbacEnforcer *RBACEnforcer
		hookFunction httpProxyHookFunction
	}
)

var httpProxyHookMap map[string]*httpProxyHook

func init() {
	httpProxyHookMap = make(map[string]*httpProxyHook, 0)
}

func AddProxyHook(hookFunc httpProxyHookFunction, rbacEnforcer *RBACEnforcer) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	file = filepath.ToSlash(file)
	_, after, found := strings.Cut(file, "/proxys/")
	if !found {
		return
	}

	if nil == rbacEnforcer {
		rbacEnforcer = &RBACEnforcer{}
	}

	after = strings.TrimSuffix(after, ".go")
	httpProxyHookMap[after] = &httpProxyHook{
		rbacEnforcer: rbacEnforcer,
		hookFunction: hookFunc,
	}
}

type RBACEnforcer struct {
	authRequired bool

	requireMatchAll []string
	requireMatchAny []string
	denyMatchAll    []string
	denyMatchAny    []string
}

func (e *RBACEnforcer) Enforce(r *base.HttpTransportHookRequest) (proceed bool) {
	if !e.authRequired {
		return true
	}
	user := r.User
	if user == nil {
		return false
	}
	if ok := e.enforceRequireMatchAll(user); !ok {
		return false
	}
	if ok := e.enforceRequireMatchAny(user); !ok {
		return false
	}
	if ok := e.enforceDenyMatchAll(user); !ok {
		return false
	}
	if ok := e.enforceDenyMatchAny(user); !ok {
		return false
	}
	return true
}

func (e *RBACEnforcer) enforceRequireMatchAll(user *base.WunderGraphUser[string]) bool {
	if len(e.requireMatchAll) == 0 {
		return true
	}
	for _, match := range e.requireMatchAll {
		if contains := e.containsOne(user.Roles, match); !contains {
			return false
		}
	}
	return true
}

func (e *RBACEnforcer) enforceRequireMatchAny(user *base.WunderGraphUser[string]) bool {
	if len(e.requireMatchAny) == 0 {
		return true
	}
	for _, match := range e.requireMatchAny {
		if contains := e.containsOne(user.Roles, match); contains {
			return true
		}
	}
	return false
}

func (e *RBACEnforcer) enforceDenyMatchAll(user *base.WunderGraphUser[string]) bool {
	if len(e.denyMatchAll) == 0 {
		return true
	}
	for _, match := range e.denyMatchAll {
		if contains := e.containsOne(user.Roles, match); !contains {
			return true
		}
	}
	return false
}

func (e *RBACEnforcer) enforceDenyMatchAny(user *base.WunderGraphUser[string]) bool {
	if len(e.denyMatchAny) == 0 {
		return true
	}
	for _, match := range e.denyMatchAny {
		if contains := e.containsOne(user.Roles, match); contains {
			return false
		}
	}
	return true
}

func (e *RBACEnforcer) containsOne(slice []string, one string) bool {
	for i := range slice {
		if slice[i] == one {
			return true
		}
	}
	return false
}

func RegisterProxyHooks(e *echo.Echo) {
	apiPrefixPath := "/proxy"
	for name, proxyHook := range httpProxyHookMap {
		apiPath := path.Join(apiPrefixPath, name)
		e.Logger.Debugf(`Registered proxyHook [%s]`, apiPath)
		e.POST(apiPath, func(c echo.Context) error {
			brc := c.(*base.HttpTransportHookRequest)
			if proceed := proxyHook.rbacEnforcer.Enforce(brc); !proceed {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
			}

			var reqBody HttpTransportBody
			err := c.Bind(&reqBody)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}

			newResp, err := proxyHook.hookFunction(brc, &reqBody)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
			resp := map[string]interface{}{
				"op":       reqBody.Name,
				"hook":     "proxyHook",
				"response": map[string]interface{}{},
			}
			if newResp != nil {
				resp["response"].(map[string]interface{})["response"] = newResp
			}
			return c.JSON(http.StatusOK, resp)
		})
	}
}
