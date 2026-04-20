package middleware

import (
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func metadataUUID(md metadata.MD, key string) (uuid.UUID, error) {
	values := md.Get(key)
	if len(values) == 0 {
		return uuid.Nil, status.Errorf(codes.Unauthenticated, "%s metadata is required", key)
	}

	id, err := uuid.Parse(values[0])
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, fmt.Sprintf("%s must be a valid UUID", key))
	}
	return id, nil
}
