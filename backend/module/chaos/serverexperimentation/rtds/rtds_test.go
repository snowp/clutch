package rtds

import (
	"context"
	"net"
	"testing"
	"time"

	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	gcpDiscoveryV2 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	gcpDiscoveryV3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	gcpRuntimeServiceV3 "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"go.uber.org/zap"
	rpc_status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	rtdsconfigv1 "github.com/lyft/clutch/backend/api/config/module/chaos/experimentation/rtds/v1"
	"github.com/lyft/clutch/backend/module/moduletest"
	"github.com/lyft/clutch/backend/service"
	"github.com/lyft/clutch/backend/service/chaos/experimentation/experimentstore"
)

type testServer struct {
	registrar *moduletest.TestRegistrar
	scope     tally.TestScope
	storer    *simpleStorer
}

func newTestServer(t *testing.T, ttl bool) testServer {
	t.Helper()
	server := testServer{}

	server.storer = &simpleStorer{}
	service.Registry[experimentstore.Name] = server.storer

	// Set up a test server listening to :9000.
	config := &rtdsconfigv1.Config{
		RtdsLayerName:             "tests",
		CacheRefreshInterval:      ptypes.DurationProto(time.Second),
		IngressFaultRuntimePrefix: "ingress",
		EgressFaultRuntimePrefix:  "egress",
	}

	if ttl {
		config.ResourceTtl = &durationpb.Duration{
			Seconds: 1,
		}
		config.HeartbeatInterval = &durationpb.Duration{
			Seconds: 1,
		}
	}

	any, err := ptypes.MarshalAny(config)
	assert.NoError(t, err)
	zap.NewNop()
	server.scope = tally.NewTestScope("test", nil)

	m, err := New(any, zap.NewNop(), server.scope)
	assert.NoError(t, err)

	server.registrar = moduletest.NewRegisterChecker()

	err = m.Register(server.registrar)
	assert.NoError(t, err)

	l, err := net.Listen("tcp", "localhost:9000")
	assert.NoError(t, err)

	go func() {
		err := server.registrar.GRPCServer().Serve(l)
		assert.NoError(t, err)
	}()

	return server
}

func (t *testServer) stop() {
	t.registrar.GRPCServer().Stop()
}

func (t *testServer) clientConn() (*grpc.ClientConn, error) {
	return grpc.Dial("localhost:9000", grpc.WithInsecure())
}

func TestServerStats(t *testing.T) {
	testServer := newTestServer(t, false)
	defer testServer.stop()

	// Connect to the test server.
	conn, err := testServer.clientConn()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Verify V3 stats.
	v3Client := gcpRuntimeServiceV3.NewRuntimeDiscoveryServiceClient(conn)
	v3Stream, err := v3Client.StreamRuntime(ctx)
	assert.NoError(t, err)
	defer func() {
		err := v3Stream.CloseSend()
		assert.NoError(t, err)
	}()

	// Regular flow.
	err = v3Stream.Send(&gcpDiscoveryV3.DiscoveryRequest{})
	assert.NoError(t, err)

	_, err = v3Stream.Recv()
	assert.NoError(t, err)

	assert.Equal(t, int64(1), testServer.scope.Snapshot().Counters()["test.v3.totalResourcesServed+"].Value())
	assert.Equal(t, int64(0), testServer.scope.Snapshot().Counters()["test.v3.totalErrorsReceived+"].Value())

	// Error response from xDS client.
	err = v3Stream.Send(&gcpDiscoveryV3.DiscoveryRequest{ErrorDetail: &rpc_status.Status{}})
	assert.NoError(t, err)

	assert.Equal(t, int64(1), testServer.scope.Snapshot().Counters()["test.v3.totalResourcesServed+"].Value())
	// Async verification here since it appears that we don't get a response back in this case, so we
	// aren't able to synchronize on the response.
	awaitCounterEquals(t, testServer.scope, "test.v3.totalErrorsReceived+", 1)

	// Verify V2 stats.
	v2Client := gcpDiscoveryV2.NewRuntimeDiscoveryServiceClient(conn)
	v2Stream, err := v2Client.StreamRuntime(ctx)
	assert.NoError(t, err)
	defer func() {
		err := v3Stream.CloseSend()
		assert.NoError(t, err)
	}()

	// Regular flow.
	err = v2Stream.Send(&envoy_api_v2.DiscoveryRequest{})
	assert.NoError(t, err)

	_, err = v2Stream.Recv()
	assert.NoError(t, err)

	assert.Equal(t, int64(1), testServer.scope.Snapshot().Counters()["test.v2.totalResourcesServed+"].Value())

	// Error response from xDS client.
	err = v2Stream.Send(&envoy_api_v2.DiscoveryRequest{ErrorDetail: &rpc_status.Status{}})
	assert.NoError(t, err)

	assert.Equal(t, int64(1), testServer.scope.Snapshot().Counters()["test.v2.totalResourcesServed+"].Value())
	// Async verification here since it appears that we don't get a response back in this case, so we
	// aren't able to synchronize on the response.
	awaitCounterEquals(t, testServer.scope, "test.v2.totalErrorsReceived+", 1)
}

func awaitCounterEquals(t *testing.T, scope tally.TestScope, counter string, value int64) {
	t.Helper()

	timeout := time.NewTimer(time.Second)
	for {
		v := int64(0)
		c, ok := scope.Snapshot().Counters()[counter]
		if ok {
			v = c.Value()
		}

		if value == v {
			return
		}

		select {
		case <-timeout.C:
			t.Errorf("timed out waiting for %s to become %d, last value %d", counter, value, v)
			return
		case <-time.After(100 * time.Millisecond):
			break
		}
	}
}

// Verifies that TTL and heartbeating is done when configured to do so.
func TestResourceTTL(t *testing.T) {
	testServer := newTestServer(t, true)
	defer testServer.stop()

	conn, err := testServer.clientConn()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := gcpRuntimeServiceV3.NewRuntimeDiscoveryServiceClient(conn)
	stream, err := client.StreamRuntime(ctx)
	assert.NoError(t, err)
	defer func() {
		err := stream.CloseSend()
		assert.NoError(t, err)
	}()

	// Flow:
	// Send initial request
	// Receive response
	// Send request responding to response
	// Receive heartbeat response
	err = stream.Send(&gcpDiscoveryV3.DiscoveryRequest{})
	assert.NoError(t, err)

	now := time.Now()
	testServer.storer.CreateExperiment(context.Background(), &anypb.Any{}, &now, &now)
	r, err := stream.Recv()
	assert.NoError(t, err)

	resource := &gcpDiscoveryV3.Resource{}
	assert.NoError(t, ptypes.UnmarshalAny(r.Resources[0], resource))

	assert.Equal(t, int64(1), resource.Ttl.Seconds)

	err = stream.Send(&gcpDiscoveryV3.DiscoveryRequest{
		VersionInfo:   r.VersionInfo,
		ResponseNonce: r.Nonce,
	})
	assert.NoError(t, err)

	r, err = stream.Recv()
	assert.NoError(t, err)

	resource = &gcpDiscoveryV3.Resource{}
	assert.NoError(t, ptypes.UnmarshalAny(r.Resources[0], resource))

	assert.Equal(t, int64(1), resource.Ttl.Seconds)
	assert.Nil(t, resource.Resource)
}
