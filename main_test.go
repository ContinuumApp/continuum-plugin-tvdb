package main

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	pluginv1 "github.com/ContinuumApp/continuum-plugin-sdk/pkg/pluginproto/continuum/plugin/v1"
)

func TestRuntimeServerConfigure_ConfiguresTVDBProvider(t *testing.T) {
	server := &runtimeServer{}

	_, err := server.Configure(context.Background(), &pluginv1.ConfigureRequest{
		Config: []*pluginv1.ConfigEntry{
			{
				Key: "connection",
				Value: mustStruct(t, map[string]any{
					"api_key": "tvdb-api-key",
					"pin":     "tvdb-pin",
				}),
			},
		},
	})
	if err != nil {
		t.Fatalf("Configure() returned error: %v", err)
	}

	provider, err := server.providerForRequest()
	if err != nil {
		t.Fatalf("providerForRequest() returned error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider to be configured")
	}
	if server.config.APIKey != "tvdb-api-key" {
		t.Fatalf("config.APIKey = %q, want tvdb-api-key", server.config.APIKey)
	}
	if server.config.PIN != "tvdb-pin" {
		t.Fatalf("config.PIN = %q, want tvdb-pin", server.config.PIN)
	}
}

func TestRuntimeServerConfigure_RequiresTVDBCredentials(t *testing.T) {
	server := &runtimeServer{}

	_, err := server.Configure(context.Background(), &pluginv1.ConfigureRequest{})
	if err != nil {
		t.Fatalf("Configure() returned error: %v", err)
	}

	_, err = server.providerForRequest()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("providerForRequest() error code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func mustStruct(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	if err != nil {
		t.Fatalf("structpb.NewStruct() returned error: %v", err)
	}
	return result
}
