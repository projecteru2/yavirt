package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	engineapi "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
	pkgtypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
	"github.com/projecteru2/yavirt/pkg/vmimage/utils"
)

const (
	destImgName      = "vm.img"
	dockerCliVersion = "1.35"
)

type Manager struct {
	cfg *pkgtypes.Config
	cli *engineapi.Client
}

func NewManager(config *pkgtypes.Config) (m *Manager, err error) {
	cli, err := makeDockerClient(config.Docker.Endpoint)
	if err != nil {
		return nil, err
	}
	m = &Manager{
		cfg: config,
		cli: cli,
	}
	return m, nil
}

func (mgr *Manager) ListLocalImages(ctx context.Context, user string) ([]*pkgtypes.Image, error) {
	images, err := mgr.cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	var ans []*pkgtypes.Image
	prefix := path.Join(mgr.cfg.Docker.Prefix, user)
	for _, dockerImg := range images {
		for _, repoTag := range dockerImg.RepoTags {
			if strings.HasPrefix(repoTag, prefix) {
				fullname := strings.TrimPrefix(repoTag, prefix)
				fullname = strings.TrimPrefix(fullname, "/")
				fullname = strings.TrimPrefix(fullname, "library/")
				img, _ := pkgtypes.NewImage(fullname)
				ans = append(ans, img)
			}
		}
	}
	return ans, nil
}

func (mgr *Manager) LoadImage(ctx context.Context, imgName string) (img *pkgtypes.Image, err error) {
	if img, err = pkgtypes.NewImage(imgName); err != nil {
		return nil, err
	}
	rc, err := mgr.Pull(ctx, img, pkgtypes.PullPolicyAlways)
	if err != nil {
		return nil, err
	}
	if err := utils.EnsureReaderClosed(rc); err != nil {
		return nil, err
	}
	if err := mgr.loadMetadata(ctx, img); err != nil {
		return nil, err
	}
	return img, nil
}

// Prepare prepares the image for use by creating a Dockerfile and building a Docker image.
//
// Parameters:
//   - fname: a local filename or an url
//
// Returns:
//   - io.ReadCloser: a ReadCloser to read the prepared image.
//   - error: an error if any occurred during the preparation process.
func (mgr *Manager) Prepare(ctx context.Context, fname string, img *pkgtypes.Image) (io.ReadCloser, error) {
	cli := mgr.cli
	baseDir := filepath.Dir(fname)
	baseName := filepath.Base(fname)
	digest := ""
	tarOpts := &archive.TarOptions{
		IncludeFiles: []string{baseName, "Dockerfile.yavirt"},
		Compression:  archive.Uncompressed,
		NoLchown:     true,
	}
	if u, err := url.Parse(fname); err == nil && u.Scheme != "" && u.Host != "" {
		tmpDir, err := os.MkdirTemp(os.TempDir(), "image-prepare-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpDir)
		baseDir = tmpDir
		baseName = fname
		tarOpts.IncludeFiles = []string{"Dockerfile.yavirt"}
		if digest, err = httpGetSHA256(ctx, fname); err != nil {
			return nil, err
		}
	} else {
		if digest, err = utils.CalcDigestOfFile(fname); err != nil {
			return nil, err
		}
	}
	dockerfile := fmt.Sprintf("FROM scratch\nLABEL SHA256=%s\nADD %s /%s", digest, baseName, destImgName)
	if err := os.WriteFile(filepath.Join(baseDir, "Dockerfile.yavirt"), []byte(dockerfile), 0600); err != nil {
		return nil, err
	}
	defer os.Remove(filepath.Join(baseDir, "Dockerfile.yavirt"))

	// Create a build context from the specified directory
	buildContext, err := archive.TarWithOptions(baseDir, tarOpts)
	if err != nil {
		return nil, err
	}

	// Build the Docker image using the build context
	buildOptions := types.ImageBuildOptions{
		Context:    buildContext,
		Dockerfile: "Dockerfile.yavirt", // Use the default Dockerfile name
		Tags:       []string{mgr.dockerImageName(img)},
	}

	resp, err := cli.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (mgr *Manager) Pull(ctx context.Context, img *pkgtypes.Image, _ pkgtypes.PullPolicy) (io.ReadCloser, error) {
	cli, cfg := mgr.cli, mgr.cfg
	return cli.ImagePull(ctx, mgr.dockerImageName(img), types.ImagePullOptions{
		RegistryAuth: cfg.Docker.Auth,
	})
}

func (mgr *Manager) Push(ctx context.Context, img *pkgtypes.Image, force bool) (io.ReadCloser, error) {
	cli, cfg := mgr.cli, mgr.cfg
	return cli.ImagePush(ctx, mgr.dockerImageName(img), types.ImagePushOptions{
		RegistryAuth: cfg.Docker.Auth,
		All:          force,
	})
}

func (mgr *Manager) RemoveLocal(ctx context.Context, img *pkgtypes.Image) error {
	cli := mgr.cli
	_, err := cli.ImageRemove(ctx, mgr.dockerImageName(img), types.ImageRemoveOptions{
		Force:         true, // Remove even if the image is in use
		PruneChildren: true, // Prune dependent child images
	})
	return err
}

func (mgr *Manager) CheckHealth(_ context.Context) error {
	parts := strings.SplitN(mgr.cfg.Docker.Prefix, "/", 2)
	if len(parts) >= 2 && strings.Contains(parts[0], ".") {
		if err := utils.IPReachable(parts[0], time.Second); err != nil {
			return errors.Wrapf(err, "failed to ping image hub %s", parts[0])
		}
	}
	return nil
}
func (mgr *Manager) loadMetadata(ctx context.Context, img *pkgtypes.Image) (err error) {
	cli := mgr.cli
	resp, _, err := cli.ImageInspectWithRaw(ctx, mgr.dockerImageName(img))
	if err != nil {
		return err
	}
	upperDir := resp.GraphDriver.Data["UpperDir"]
	img.LocalPath = filepath.Join(upperDir, destImgName)
	img.ActualSize, img.VirtualSize, err = utils.ImageSize(ctx, img.LocalPath)

	img.Digest = resp.Config.Labels["SHA256"]
	return err
}

func (mgr *Manager) dockerImageName(img *pkgtypes.Image) string {
	cfg := mgr.cfg
	if img.Username == "" {
		return path.Join(cfg.Docker.Prefix, "library", img.Fullname())
	} else { //nolint
		return path.Join(cfg.Docker.Prefix, img.Fullname())
	}
}

func makeDockerClient(endpoint string) (*engineapi.Client, error) {
	defaultHeaders := map[string]string{"User-Agent": "eru-yavirt"}
	return engineapi.NewClientWithOpts(
		engineapi.WithHost(endpoint),
		engineapi.WithVersion(dockerCliVersion),
		engineapi.WithHTTPClient(nil),
		engineapi.WithHTTPHeaders(defaultHeaders))
}

func httpGetSHA256(ctx context.Context, u string) (string, error) {
	if !strings.HasSuffix(u, ".img") {
		return "", fmt.Errorf("invalid url: %s", u)
	}
	url := strings.TrimSuffix(u, ".img")
	url += ".sha256sum"
	// Perform GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
