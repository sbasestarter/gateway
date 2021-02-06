package gateway

import (
	"strings"
	"sync"

	"github.com/jiuzhou-zhao/go-fundamental/clienttoolset"
	"github.com/jiuzhou-zhao/go-fundamental/dbtoolset"
	"github.com/jiuzhou-zhao/go-fundamental/grpce"
	"github.com/jiuzhou-zhao/go-fundamental/grpce/meta"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/sbasestarter/gateway/internal/config"
	"github.com/sgostarter/librediscovery"
	"github.com/trusch/grpc-proxy/proxy"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	gRpcSchema = "grpclb"
)

type serviceWithConn struct {
	conn *grpc.ClientConn
}

type Gateway struct {
	sync.RWMutex
	ctx context.Context
	cfg *config.Config

	serviceWithConns map[string]*serviceWithConn
}

func NewGateway(ctx context.Context, cfg *config.Config, dbToolset *dbtoolset.DBToolset) *Gateway {
	getter, err := librediscovery.NewDefaultGetter(ctx, dbToolset.GetRedis())
	if err != nil {
		loge.Fatalf(ctx, "new discovery getter failed: %v", err)
		return nil
	}

	err = clienttoolset.RegisterSchemas(ctx, &clienttoolset.RegisterSchemasConfig{
		Getter:  getter,
		Logger:  loge.GetGlobalLogger().GetLogger(),
		Schemas: []string{gRpcSchema},
	})
	if err != nil {
		loge.Fatalf(ctx, "register schema failed: %v", err)
		return nil
	}

	gw := &Gateway{
		ctx:              ctx,
		cfg:              cfg,
		serviceWithConns: make(map[string]*serviceWithConn),
	}

	return gw
}

func (gw *Gateway) getGRpcServiceInfoOnCache(gRpcClass string) (conn *grpc.ClientConn, err error) {
	gw.RLock()
	defer gw.RUnlock()

	sc, ok := gw.serviceWithConns[gRpcClass]
	if !ok {
		return nil, nil
	}
	return sc.conn, nil
}

func (gw *Gateway) getGRpcServiceInfo(gRpcClass string) (conn *grpc.ClientConn, err error) {
	address := grpce.GetDialAddressByGRpcClassName(gRpcClass)
	if address == "" {
		err = status.Error(codes.NotFound, gRpcClass)
		return
	}
	cfg := gw.cfg.GRpcClientConfigTpl
	cfg.Address = address
	conn, err = clienttoolset.DialGRpcServer(&cfg, nil)
	if err != nil {
		loge.Errorf(gw.ctx, "dial %v failed: %v, %+v", address, err, cfg)
		return
	}
	gw.Lock()
	defer gw.Unlock()
	gw.serviceWithConns[gRpcClass] = &serviceWithConn{
		conn: conn,
	}
	return
}

func (gw *Gateway) GetGRpcDirector() proxy.StreamDirector {
	return func(ctx context.Context, fullMethodName string) (context context.Context, conn *grpc.ClientConn, err error) {
		context = meta.TransferContextMeta(ctx, nil)

		clsName := fullMethodName[:strings.Index(fullMethodName[1:], "/")+1]
		conn, err = gw.getGRpcServiceInfoOnCache(clsName)
		if err != nil {
			return
		}
		if conn != nil {
			return
		}
		conn, err = gw.getGRpcServiceInfo(clsName)
		return
	}
}
