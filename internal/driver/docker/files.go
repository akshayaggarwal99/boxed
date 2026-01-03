package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/akshayaggarwal99/boxed/internal/driver"
	"github.com/docker/docker/api/types"
)

// ListFiles implements driver.Driver.
func (d *DockerDriver) ListFiles(ctx context.Context, id, path string) ([]*driver.FileEntry, error) {
	absPath, err := d.resolvePath(ctx, id, path)
	if err != nil {
		return nil, err
	}

	// We use CopyFromContainer to get a tar stream of the path.
	reader, _, err := d.cli.CopyFromContainer(ctx, id, absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read path: %w", err)
	}
	defer reader.Close()

	tr := tar.NewReader(reader)
	var entries []*driver.FileEntry

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}

		name := header.Name
		name = strings.TrimPrefix(name, "/")

		entry := &driver.FileEntry{
			Name:         filepath.Base(name),
			Path:         name,
			Size:         header.Size,
			Mode:         header.Mode,
			IsDir:        header.Typeflag == tar.TypeDir,
			LastModified: header.ModTime,
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// PutFile implements driver.Driver.
func (d *DockerDriver) PutFile(ctx context.Context, id, path string, content io.Reader) error {
	absPath, err := d.resolvePath(ctx, id, path)
	if err != nil {
		return err
	}

	// Create a tar stream
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read content: %w", err)
	}

	header := &tar.Header{
		Name:    filepath.Base(absPath),
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("tar write header failed: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("tar write body failed: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar close failed: %w", err)
	}

	// CopyToContainer expects the path to be the directory *containing* the file.
	dir := filepath.Dir(absPath)

	err = d.cli.CopyToContainer(ctx, id, dir, &buf, types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("docker copy failed: %w", err)
	}
	return nil
}

// GetFile implements driver.Driver.
func (d *DockerDriver) GetFile(ctx context.Context, id, path string) (io.ReadCloser, error) {
	absPath, err := d.resolvePath(ctx, id, path)
	if err != nil {
		return nil, err
	}

	reader, _, err := d.cli.CopyFromContainer(ctx, id, absPath)
	if err != nil {
		return nil, fmt.Errorf("docker copy failed: %w", err)
	}

	// The reader is a Tar stream. We need to extract the single file content.
	tr := tar.NewReader(reader)

	// Advance to first entry
	_, err = tr.Next()
	if err != nil {
		reader.Close()
		return nil, fmt.Errorf("file not found in tar: %w", err)
	}

	return &tarReadCloser{tr: tr, closer: reader}, nil
}

func (d *DockerDriver) resolvePath(ctx context.Context, id, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	info, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	workDir := info.Config.WorkingDir
	if workDir == "" {
		workDir = "/"
	}
	return filepath.Join(workDir, path), nil
}

type tarReadCloser struct {
	tr     *tar.Reader
	closer io.Closer
}

func (t *tarReadCloser) Read(p []byte) (int, error) {
	return t.tr.Read(p)
}

func (t *tarReadCloser) Close() error {
	return t.closer.Close()
}
