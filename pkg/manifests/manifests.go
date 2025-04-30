package manifests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/moeru-ai/mcp-launcher/internal/contexts"
	"github.com/moeru-ai/mcp-launcher/pkg/jsonpatch"
	"github.com/nekomeowww/xo"
	"github.com/samber/mo"
	"gopkg.in/yaml.v3"
)

// ManifestOperation defines the type of operation to perform
type OperationType string

const (
	OperationPatchJSON OperationType = "PatchJson"
	// Add more operation types as needed
)

// Match represents a rule that determines if a manifest should be applied
type Match struct {
	Type      string   `yaml:"type" json:"type"`
	Value     string   `yaml:"value" json:"value"`
	ExactURLs []string `yaml:"exactUrls,omitempty" json:"exactUrls,omitempty"`
}

// JSONPatchOperation defines a JSON patch operation
type JSONPatchOperation struct {
	Op    string      `yaml:"op" json:"op"`
	Path  string      `yaml:"path" json:"path"`
	Value interface{} `yaml:"value,omitempty" json:"value,omitempty"`
}

// Operation represents a single transformation operation
type Operation struct {
	On         string               `yaml:"on" json:"on"`
	Type       OperationType        `yaml:"type" json:"type"`
	TargetPath string               `yaml:"targetPath" json:"targetPath"`
	Patches    []JSONPatchOperation `yaml:"patches,omitempty" json:"patches,omitempty"`
}

type ManifestSpec struct {
	Match      Match       `yaml:"match" json:"match"`
	Operations []Operation `yaml:"operations" json:"operations"`
}

type ManifestDef struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
}

type ManifestMeta struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
}

// Manifest represents a configuration for repository transformations
type Manifest struct {
	ManifestDef

	Metadata ManifestMeta `yaml:"metadata" json:"metadata"`
	Spec     ManifestSpec `yaml:"spec" json:"spec"`
}

// ManifestRegistry manages loading and execution of manifests
type ManifestRegistry struct {
	manifests []Manifest
}

// NewManifestRegistry creates a new registry
func NewManifestRegistry() *ManifestRegistry {
	return &ManifestRegistry{
		manifests: []Manifest{},
	}
}

// LoadManifestsFromDirectory loads all manifest files from a directory
func (r *ManifestRegistry) LoadManifestsFromDirectory(ctx context.Context, dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			return r.LoadManifestsFromDirectory(ctx, filepath.Join(dirPath, file.Name()))
		}

		ext := filepath.Ext(file.Name())
		if ext != ".yaml" && ext != ".yml" && ext != ".json" && ext != ".toml" {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())

		manifest, err := LoadManifestFromFile(filePath)
		if err != nil {
			contexts.SlogFrom(ctx).Error("Failed to load manifest", "file", filePath, "error", err)
			continue
		}

		r.manifests = append(r.manifests, manifest)

		contexts.SlogFrom(ctx).Info("Loaded manifest", "name", manifest.Metadata.Name)
	}

	return nil
}

// LoadManifestFromFile loads a manifest from a file
func LoadManifestFromFile(filePath string) (Manifest, error) {
	var manifest Manifest

	data, err := os.ReadFile(filePath)
	if err != nil {
		return manifest, err
	}

	// Here you'd add logic to detect file type (yaml/json/toml) and parse accordingly
	// For simplicity, this example assumes YAML
	err = yaml.Unmarshal(data, &manifest)
	if err != nil {
		return manifest, err
	}

	return manifest, nil
}

// ExecuteManifests runs all applicable manifests in the current context
func (r *ManifestRegistry) ExecuteManifests(ctx context.Context, phase string) error {
	md := contexts.MetadataFrom(ctx)

	for _, manifest := range r.manifests {
		if shouldApplyManifest(manifest.Spec.Match, md) {
			contexts.SlogFrom(ctx).Info("Executing manifest", "name", manifest.Metadata.Name, "phase", phase)

			err := executeManifest(ctx, manifest, phase)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// shouldApplyManifest checks if a manifest should be applied based on conditions
func shouldApplyManifest(condition Match, md *contexts.Metadata) bool {
	if condition.Type == "RepositoryUrl" {
		for _, url := range condition.ExactURLs {
			if url == md.RepositoryURL {
				return true
			}
		}
	}

	return false
}

// executeManifest runs the operations defined in a manifest
func executeManifest(ctx context.Context, manifest Manifest, on string) error {
	md := contexts.MetadataFrom(ctx)

	for _, op := range manifest.Spec.Operations {
		if op.On != on {
			continue
		}

		switch op.Type {
		case OperationPatchJSON:
			targetPath := filepath.Join(md.RepositoryClonedPath, md.SubDirectory, op.TargetPath)

			content, err := os.ReadFile(targetPath)
			if err != nil {
				return err
			}

			// Convert manifest patch operations to jsonpatch operations
			patches := make([]mo.Option[jsonpatch.JSONPatchOperationObject], 0, len(op.Patches))

			for _, p := range op.Patches {
				switch p.Op {
				case "add":
					patches = append(patches, jsonpatch.NewAdd(p.Path, p.Value))
				case "remove":
					patches = append(patches, jsonpatch.NewRemove(p.Path))
				}
			}

			patchedContent := jsonpatch.ApplyPatches(
				content,
				mo.Some(jsonpatch.ApplyOptions{AllowMissingPathOnRemove: true}),
				patches...,
			)

			if patchedContent.IsError() {
				return patchedContent.Error()
			}

			contexts.SlogFrom(ctx).Info("Patching file", "path", targetPath)

			err = os.WriteFile(targetPath, patchedContent.MustGet(), 0600) //nolint:mnd
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown operation type: %s", op.Type)
		}
	}

	return nil
}

var DefaultRegistry *ManifestRegistry

func init() {
	DefaultRegistry = NewManifestRegistry()
}

func LoadManifests(ctx context.Context) error {
	err := DefaultRegistry.LoadManifestsFromDirectory(ctx, xo.RelativePathBasedOnPwdOf("manifests"))
	if err != nil {
		contexts.SlogFrom(ctx).Warn("Failed to load manifests from default directory", "error", err)
	}

	if envPath := os.Getenv("MCP_MANIFESTS_PATH"); envPath != "" {
		err := DefaultRegistry.LoadManifestsFromDirectory(ctx, envPath)
		if err != nil {
			contexts.SlogFrom(ctx).Warn("Failed to load manifests from environment path", "path", envPath, "error", err)
		}
	}

	return nil
}

func ExecuteAfterClone(ctx context.Context) error {
	return DefaultRegistry.ExecuteManifests(ctx, "AfterClone")
}

func ExecuteBeforeBuild(ctx context.Context) error {
	return DefaultRegistry.ExecuteManifests(ctx, "BeforeBuild")
}
