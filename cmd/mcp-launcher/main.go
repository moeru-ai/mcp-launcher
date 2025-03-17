package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"os/exec"
	"path/filepath"
	"strings"
	"time"

	dockerfile "github.com/flexstack/new-dockerfile"
	"github.com/go-git/go-git/v5"
	"github.com/lmittmann/tint"
	"github.com/moby/buildkit/client"
	"github.com/moeru-ai/mcp-launcher/internal/plugins"
	"github.com/moeru-ai/mcp-launcher/pkg/pluginregistry"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func getXDGDataHome() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return xdg
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".local", "share")
}

func parseRepoURL(url string) (string, error) {
	// Handle both HTTPS and SSH URLs
	if strings.HasPrefix(url, "https://github.com/") {
		return strings.TrimPrefix(url, "https://"), nil
	}
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		return "github.com/" + strings.TrimSuffix(path, ".git"), nil
	}

	return "", fmt.Errorf("unsupported repository URL format: %s", url)
}

func isWorkingTreeClean(repo *git.Repository) (bool, error) {
	w, err := repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree status: %w", err)
	}

	return status.IsClean(), nil
}

func resetHard(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Reset to HEAD
	err = w.Reset(&git.ResetOptions{
		Commit: ref.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to reset to HEAD: %w", err)
	}

	return nil
}

func pullLatest(repo *git.Repository) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
		Force:      false,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to pull latest changes: %w", err)
	}

	return nil
}

func cloneRepository(repoURL string) (string, error) {
	repoPath, err := parseRepoURL(repoURL)
	if err != nil {
		return "", err
	}

	// Create the full path for the repository
	targetPath := filepath.Join(getXDGDataHome(), "mcp-launcher", "servers", "source", repoPath)

	// Ensure the parent directory exists
	err = os.MkdirAll(filepath.Dir(targetPath), 0755) //nolint:mnd
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	var repo *git.Repository

	// Check if repository already exists
	if _, err := os.Stat(filepath.Join(targetPath, ".git")); err == nil {
		// Repository exists, open it
		repo, err = git.PlainOpen(targetPath)
		if err != nil {
			return "", fmt.Errorf("failed to open existing repository: %w", err)
		}

		// Check if working tree is clean
		clean, err := isWorkingTreeClean(repo)
		if err != nil {
			return "", err
		}

		if !clean {
			// Instead of stashing (which isn't supported by go-git), we'll reset to HEAD
			if err := resetHard(repo); err != nil {
				return "", err
			}
		}

		if err := pullLatest(repo); err != nil {
			return "", err
		}

		return targetPath, nil
	}

	// Clone new repository
	cloneOpts := &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Depth:    1,
	}

	_, err = git.PlainClone(targetPath, false, cloneOpts)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return targetPath, nil
}

func printStatus(status client.SolveStatus) {
	// Process vertexes (build steps)
	for _, vertex := range status.Vertexes {
		// Skip if no name
		if vertex.Name == "" {
			continue
		}

		status := "RUNNING"
		if vertex.Completed != nil {
			status = "DONE   "
		} else if vertex.Cached {
			status = "CACHED "
		}

		// Extract step number from name if available
		stepInfo := ""
		if strings.Contains(vertex.Name, "] ") {
			parts := strings.SplitN(vertex.Name, "] ", 2) //nolint:mnd
			if len(parts) == 2 {                          //nolint:mnd
				stepInfo = parts[0] + "] "
				vertex.Name = parts[1]
			}
		}

		fmt.Printf("\r[%s] %s%s\n", status, stepInfo, vertex.Name) //nolint:forbidigo
	}
}

var (
	directory string
)

func main() {
	cmd := &cobra.Command{
		Use:   "mcp-launcher [repository-url]",
		Short: "Clone a repository and build its Docker image",
		Args:  cobra.ExactArgs(1),
		Run: func(cobraCmd *cobra.Command, args []string) {
			level := slog.LevelInfo
			if os.Getenv("DEBUG") != "" {
				level = slog.LevelDebug
			}

			handler := tint.NewHandler(os.Stderr, &tint.Options{
				Level:      level,
				TimeFormat: time.Kitchen,
			})

			log := slog.New(handler)

			plugins.RegisterPlugins()

			ctx := context.WithValue(context.Background(), "repoURL", args[0])

			err := pluginregistry.BeforeClone(ctx)
			if err != nil {
				log.Error("Failed to run before clone plugins", slog.Any("error", err))
				os.Exit(1)
			}

			// Clone the repository
			repoPath, err := cloneRepository(args[0])
			if err != nil {
				log.Error("Failed to clone repository", slog.Any("error", err))
				os.Exit(1)
			}

			log.Info("Repository cloned successfully", slog.String("path", repoPath))

			// Add debug information about the repository structure
			if err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				relPath, err := filepath.Rel(repoPath, path)
				if err != nil {
					return err
				}
				log.Debug("Found file",
					slog.String("path", relPath),
					slog.Bool("is_dir", info.IsDir()),
					slog.Int64("size", info.Size()))
				return nil
			}); err != nil {
				log.Error("Failed to walk repository", slog.Any("error", err))
			}

			ctx = context.WithValue(ctx, "clonedPath", repoPath)
			if directory != "" {
				ctx = context.WithValue(ctx, "directory", directory)
			}

			err = pluginregistry.AfterClone(ctx)
			if err != nil {
				log.Error("Failed to run after clone plugins", slog.Any("error", err))
				os.Exit(1)
			}

			df := dockerfile.New(log)

			if directory != "" {
				repoPath = filepath.Join(repoPath, directory)
				log.Info("Using subdirectory", slog.String("path", repoPath))
			}

			// Use the cloned repository path instead of current working directory
			r, err := df.MatchRuntime(repoPath)
			if err != nil {
				log.Error("Failed to match runtime", slog.Any("error", err))
				os.Exit(1)
			}

			contents, err := r.GenerateDockerfile(repoPath)
			if err != nil {
				log.Error("Failed to generate Dockerfile", slog.Any("error", err))
				os.Exit(1)
			}

			log.Debug("Generated Dockerfile", slog.String("contents", string(contents)))

			tempDir, err := os.MkdirTemp("", strings.Join([]string{"mcp-launcher", "mcp-servers", "dockerfiles", "*"}, "-"))
			if err != nil {
				panic(err)
			}

			dockerfilePath := filepath.Join(tempDir, "Dockerfile")
			err = os.WriteFile(dockerfilePath, contents, 0600) //nolint:mnd
			if err != nil {
				panic(err)
			}

			log.Info("Dockerfile written", slog.String("path", dockerfilePath))

			var imageHash string

			// Create a command for Docker build
			dockerCmd := exec.Command("docker", "build", "-t", "mcp-server-dev", "-f", dockerfilePath, repoPath, "--progress=rawjson")

			log.Info("Building Docker image", slog.String("command", dockerCmd.String()))

			stderr, err := dockerCmd.StderrPipe()
			if err != nil {
				panic(err)
			}
			if err := dockerCmd.Start(); err != nil {
				panic(err)
			}

			// Process and display stdout while also parsing JSON
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()

				// Try to parse as JSON for additional processing if needed
				var data client.SolveStatus
				err := json.Unmarshal([]byte(line), &data)
				if err == nil {
					printStatus(data)

					// Check for image hash
					if len(data.Statuses) > 0 {
						imageStatus, ok := lo.Find(data.Statuses, func(item *client.VertexStatus) bool {
							return strings.HasPrefix(item.ID, "writing image")
						})
						if ok {
							imageHash = strings.TrimPrefix(imageStatus.ID, "writing image sha256:")
						}
					}
				}
			}

			if err := scanner.Err(); err != nil {
				log.Error("Error reading Docker build output", slog.Any("error", err))
			}

			// Wait for the command to complete
			if err := dockerCmd.Wait(); err != nil {
				log.Error("Docker build failed", slog.Any("error", err))
				panic(err)
			}

			err = pluginregistry.AfterBuild(ctx)
			if err != nil {
				log.Error("Failed to run after build plugins", slog.Any("error", err))
				os.Exit(1)
			}

			log.Info("Docker build completed", slog.String("dockerfile", dockerfilePath), slog.String("image_hash", imageHash))
		},
	}

	cmd.PersistentFlags().StringVarP(&directory, "directory", "d", "", "The directory that points to the entrypoint of the mcp server")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
