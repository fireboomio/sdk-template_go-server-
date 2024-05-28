package plugins

import (
	"custom-go/pkg/types"
)

var WdgHooksAndServerConfig WunderGraphHooksAndServerConfig

type (
	WunderGraphHooksAndServerConfig struct {
		Webhooks       map[string]types.WebhookConfiguration
		Hooks          HooksConfiguration
		GraphqlServers []GraphQLServerConfig
		Options        types.ServerOptions
	}
	HooksConfiguration struct {
		Global         GlobalConfiguration
		Authentication AuthenticationConfiguration
	}
)
