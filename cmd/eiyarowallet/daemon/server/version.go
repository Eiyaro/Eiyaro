package server

import (
	"context"

	"github.com/Eiyaro/Eiyaro/cmd/eiyarowallet/daemon/pb"
	"github.com/Eiyaro/Eiyaro/version"
)

func (s *server) GetVersion(_ context.Context, _ *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return &pb.GetVersionResponse{
		Version: version.Version(),
	}, nil
}
