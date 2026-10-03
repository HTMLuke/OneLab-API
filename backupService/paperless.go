package backupservice

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/HTMLuke/OneLab-API/shellService"
)

// PaperlessService creates local Paperless-ngx document exports.
type PaperlessService struct {
	dockerService shellService.ShellService
	logger        *slog.Logger
}

const (
	paperlessExportDirectory = "/tmp/export"
	localBackupDirectory     = "/tmp/onelab-paperless-backups"
)

// NewPaperlessService creates a Paperless-ngx backup source using Docker.
func NewPaperlessService(logger *slog.Logger) *PaperlessService {
	return &PaperlessService{
		dockerService: shellService.NewDockerCmdService(logger),
		logger:        logger,
	}
}

// Backup exports Paperless documents, copies the archive locally, and removes
// the temporary archive from the Paperless container.
func (s *PaperlessService) Backup(ctx context.Context) (string, error) {
	s.log(ctx, slog.LevelDebug, "paperless backup started")

	// Resolve the running Paperless webserver container before starting the export.
	containerName, err := s.resolveWebserverContainer(ctx)
	if err != nil {
		return "", err
	}

	// Create the archive inside the container and determine its generated path.
	zipPath, err := s.createExport(ctx, containerName)
	if err != nil {
		return "", err
	}

	// Copy the archive to the host so it can be encrypted or uploaded later.
	localBackupPath, err := s.copyExportLocally(ctx, containerName, zipPath)
	if err != nil {
		return "", err
	}

	// The container copy is no longer needed after the local copy succeeds.
	if err := s.removeContainerExport(ctx, containerName, zipPath); err != nil {
		return "", err
	}
	s.log(ctx, slog.LevelInfo, "paperless backup completed")
	return localBackupPath, nil
}

func (s *PaperlessService) createExport(ctx context.Context, containerName string) (string, error) {
	// Reuse a fixed temporary directory and remove stale exports first.
	if _, err := s.dockerService.Execute(ctx, containerName, "mkdir -p "+paperlessExportDirectory); err != nil {
		return "", fmt.Errorf("failed to create temp export directory in container %s: %w", containerName, err)
	}
	if _, err := s.dockerService.Execute(ctx, containerName, `sh -c "rm -f /tmp/export/export-*.zip"`); err != nil {
		return "", fmt.Errorf("failed to remove stale export files in container %s: %w", containerName, err)
	}

	// Paperless creates the ZIP using its document_exporter management command.
	output, err := s.dockerService.Execute(ctx, containerName, paperlessExportCommand())
	if err != nil {
		if output != "" {
			return "", fmt.Errorf("paperless export failed in container %s: %w: %s", containerName, err, strings.TrimSpace(output))
		}
		return "", fmt.Errorf("paperless export failed in container %s: %w", containerName, err)
	}
	listOutput, err := s.dockerService.Execute(ctx, containerName, "sh -lc 'ls -1 /tmp/export/*.zip 2>/dev/null | tail -n 1'")
	if err != nil {
		return "", fmt.Errorf("failed to locate exported zip in container %s: %w", containerName, err)
	}
	return findLatestZipInDockerExport(listOutput)
}

func (s *PaperlessService) copyExportLocally(ctx context.Context, containerName, zipPath string) (string, error) {
	archiveName := filepath.Base(zipPath)

	// Keep local backups in a dedicated temporary directory on the host.
	if err := os.MkdirAll(localBackupDirectory, 0o755); err != nil {
		return "", fmt.Errorf("failed to create local backup directory: %w", err)
	}
	localPath, err := uniqueBackupPath(localBackupDirectory, archiveName)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "docker", "cp", fmt.Sprintf("%s:%s", containerName, zipPath), localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return "", fmt.Errorf("docker cp failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("docker cp failed: %w", err)
	}
	return localPath, nil
}

func uniqueBackupPath(directory, archiveName string) (string, error) {
	path := filepath.Join(directory, archiveName)

	// Add a UTC timestamp instead of overwriting an existing backup.
	if _, err := os.Stat(path); err == nil {
		stamp := time.Now().UTC().Format("20060102-150405")
		ext := filepath.Ext(archiveName)
		name := strings.TrimSuffix(archiveName, ext)
		return filepath.Join(directory, fmt.Sprintf("%s-%s%s", name, stamp, ext)), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to inspect local backup path %s: %w", path, err)
	}
	return path, nil
}

func (s *PaperlessService) removeContainerExport(ctx context.Context, containerName, zipPath string) error {
	// Remove the temporary archive after it has been safely copied locally.
	if _, err := s.dockerService.Execute(ctx, containerName, fmt.Sprintf("rm -f -- %q", zipPath)); err != nil {
		return fmt.Errorf("failed to remove temporary export %s from container %s: %w", zipPath, containerName, err)
	}
	return nil
}

func (s *PaperlessService) resolveWebserverContainer(ctx context.Context) (string, error) {
	if s.dockerService == nil {
		return "", fmt.Errorf("docker command service is not configured")
	}

	// Ask Docker for names and IDs, then select the Paperless webserver below.
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}\t{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return "", fmt.Errorf("docker ps failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("docker ps failed: %w", err)
	}
	return findPaperlessContainerNameFromDockerPsOutput(string(output))
}

func findPaperlessContainerNameFromDockerPsOutput(output string) (string, error) {
	// Docker returns one container per line; the first field is its name.
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var candidates []string
	for _, line := range lines {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) > 0 {
			candidates = append(candidates, parts[0])
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no docker containers found")
	}

	// Prefer an exact webserver match over a generic Paperless container.
	for _, name := range candidates {
		lower := strings.ToLower(name)
		if lower == "paperless-webserver" || strings.Contains(lower, "paperless-webserver") {
			return name, nil
		}
	}
	for _, name := range candidates {
		if strings.Contains(strings.ToLower(name), "paperless") {
			return name, nil
		}
	}
	return "", fmt.Errorf("could not determine paperless webserver container from docker ps output")
}

func paperlessExportCommand() string {
	return `sh -lc '
		if [ -n "$PAPERLESS_SRC_DIR" ] && [ -f "$PAPERLESS_SRC_DIR/manage.py" ]; then
			cd "$PAPERLESS_SRC_DIR"
		elif [ -f /usr/src/paperless/src/manage.py ]; then
			cd /usr/src/paperless/src
		else
			echo "manage.py not found for paperless export" >&2
			exit 1
		fi
		python3 manage.py document_exporter /tmp/export -z
	'`
}

func findLatestZipInDockerExport(output string) (string, error) {
	// Export output can contain progress text, so keep the last ZIP-looking line.
	lines := strings.FieldsFunc(output, func(r rune) bool { return r == '\n' || r == '\r' })
	var candidate string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ".zip") {
			candidate = line
		}
	}
	if candidate == "" {
		return "", fmt.Errorf("no zip export file found in container output")
	}
	return candidate, nil
}

func (s *PaperlessService) log(ctx context.Context, level slog.Level, message string, args ...any) {
	if s.logger != nil {
		s.logger.Log(ctx, level, message, args...)
	}
}
