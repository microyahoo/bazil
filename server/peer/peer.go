package peer

import (
	"net"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server"
	"bazil.org/bazil/util/grpcedtls"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	// "google.golang.org/grpc/credentials"
	grpc_peer "google.golang.org/grpc/peer"
)

func (p *peers) auth(ctx context.Context) (*peer.PublicKey, error) {
	// https://go.googlesource.com/grpc-review/+/843cf836083053d69b704ec5a058deb437a63a28%5E1..843cf836083053d69b704ec5a058deb437a63a28/
	// authInfo, ok := credentials.FromContext(ctx)
	pr, ok := grpc_peer.FromContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	if pr.Addr == net.Addr(nil) {
		return nil, grpc.Errorf(codes.Unauthenticated, "failed to get peer address")
	}
	auth, ok := pr.AuthInfo.(*grpcedtls.Auth)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	pub := (*peer.PublicKey)(auth.PeerPub)
	getPeer := func(tx *db.Tx) error {
		_, err := tx.Peers().Get(pub)
		return err
	}
	if err := p.app.DB.View(getPeer); err != nil {
		if err == db.ErrPeerNotFound {
			return nil, grpc.Errorf(codes.PermissionDenied, "permission denied")
		}
		return nil, err
	}
	return pub, nil
}

type peers struct {
	app *server.App
}

func New(app *server.App) *grpc.Server {
	auth := &grpcedtls.Authenticator{
		Config: app.GetTLSConfig,
		// TODO Lookup:
	}
	srv := grpc.NewServer(
		grpc.Creds(auth),
	)
	rpc := &peers{app: app}
	wire.RegisterPeerServer(srv, rpc)
	return srv
}
