package contexts

import "context"

type Metadata struct {
	RepositoryURL        string
	RepositoryClonedPath string

	SubDirectory string

	DockerfilePath      string
	DockerImageHash     string
	DockerContainerHash string
}

func WithMetadata(ctx context.Context) context.Context {
	return context.WithValue(ctx, metadataKey, new(Metadata))
}

func MetadataFrom(ctx context.Context) *Metadata {
	md, ok := ctx.Value(metadataKey).(*Metadata)
	if !ok {
		return new(Metadata)
	}

	return md
}
