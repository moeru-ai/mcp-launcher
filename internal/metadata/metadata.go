package metadata

import (
	"context"
)

type Metadata struct {
	RepositoryURL        string
	RepositoryClonedPath string

	SubDirectory string

	DockerfilePath      string
	DockerImageHash     string
	DockerContainerHash string
}

// MetadataMetadataFromCtx retrieves MetadataMetadata from context
//
// NOTICE: The returned pointer allows direct access and modification of the underlying MetadataMetadata
// Be careful when modifying the properties as they are shared across the request context
func FromContext(ctx context.Context) *Metadata {
	props, pok := ctx.Value(metadataKey{}).(*metadata)
	if !pok {
		return nil
	}

	return props.request
}

func WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, metadataKey{}, &metadata{
		request: &Metadata{},
	})
}

type metadataKey struct{}

type metadata struct {
	request *Metadata
}
