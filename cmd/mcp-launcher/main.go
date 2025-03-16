package main

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	dockerfile "github.com/flexstack/new-dockerfile"
	"github.com/lmittmann/tint"
	"github.com/moby/buildkit/client"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
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
			df := dockerfile.New(log)

			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			r, err := df.MatchRuntime(wd)
			if err != nil {
				panic(err)
			}

			contents, err := r.GenerateDockerfile(wd)
			if err != nil {
				panic(err)
			}

			tempDir, err := os.MkdirTemp("", strings.Join([]string{"mcp-launcher", "mcp-servers", "dockerfiles", "*"}, "-"))
			if err != nil {
				panic(err)
			}

			dockerfilePath := filepath.Join(tempDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, contents, 0644); err != nil {
				panic(err)
			}

			var imageHash string

			// Create a command for Docker build
			dockerCmd := exec.Command("docker", "build", "-t", "mcp-server-dev", "-f", dockerfilePath, wd, "--progress=rawjson")

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
				if err := json.Unmarshal([]byte(line), &data); err == nil {
					if len(data.Statuses) > 0 {
						imageStatus, ok := lo.Find(data.Statuses, func(item *client.VertexStatus) bool { return strings.HasPrefix(item.ID, "writing image") })
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

			log.Info("Docker build completed", slog.String("dockerfile", dockerfilePath), slog.String("image_hash", imageHash))
		},
	}

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
