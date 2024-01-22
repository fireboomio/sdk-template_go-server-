package types

type (
	OperationBody[I, O any] struct {
		Op                      string                    `json:"op,omitempty"`
		Hook                    string                    `json:"hook,omitempty"`
		Input                   I                         `json:"input,omitempty"`
		Response                *OperationBodyResponse[O] `json:"response"`
		SetClientRequestHeaders map[string]string         `json:"setClientRequestHeaders,omitempty"`
	}
	OperationBodyResponse[O any] struct {
		DataAny any            `json:"dataAny,omitempty"`
		Data    O              `json:"data"`
		Errors  []GraphQLError `json:"errors"`
	}
	GraphQLError struct {
		Message string `json:"message"`
		Path    []any  `json:"path"`
	}
)

func (o *OperationBody[I, O]) ResetResponse() {
	o.Response = &OperationBodyResponse[O]{}
}

type (
	OperationHookFunction  func(hook *HookRequest, body *OperationBody[any, any]) (*OperationBody[any, any], error)
	OperationHooks         map[string]OperationConfiguration
	OperationConfiguration struct {
		MockResolve         OperationHookFunction
		PreResolve          OperationHookFunction
		PostResolve         OperationHookFunction
		MutatingPreResolve  OperationHookFunction
		MutatingPostResolve OperationHookFunction
		CustomResolve       OperationHookFunction
	}
)
