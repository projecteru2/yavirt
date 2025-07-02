package types

import (
	_ "embed"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/utils/template"

	"github.com/kdomanski/iso9660/util"

	"github.com/cockroachdb/errors"
	vmitypes "github.com/projecteru2/yavirt/pkg/vmimage/types"
)

var (
	//go:embed templates/user-data.yaml
	linuxUserData string
	//go:embed templates/meta-data.yaml
	linuxMetaData string
	//go:embed templates/network-config.yaml
	linuxNetworkData string

	//go:embed templates/windows/user-data.yaml
	winUserData string
	//go:embed templates/windows/meta-data.yaml
	winMetaData string
	//go:embed templates/windows/network-config.yaml
	winNetworkData string
)

type CloudInitGateway struct {
	IP     string `json:"ip"`
	OnLink bool   `json:"on_link"`
}

type CloudInitConfig struct {
	// use remote server to fetch cloud-init config
	URL string `json:"url"`

	// use local iso to fetch cloud-init config
	Username   string            `json:"username"`
	Password   string            `json:"password"`
	SSHPubKey  string            `json:"ssh_pub_key"`
	Hostname   string            `json:"hostname"`
	InstanceID string            `json:"instance_id"`
	Files      map[string][]byte `json:"files"`
	Commands   []string          `json:"commands"`

	MAC       string           `json:"-"`
	CIDR      string           `json:"-"`
	MTU       int              `json:"-"`
	IFName    string           `json:"-"`
	DefaultGW CloudInitGateway `json:"-"`

	OS *vmitypes.OSInfo `json:"-"`
}

func (ciCfg *CloudInitConfig) GenFilesContent() (string, string, string, error) {
	dataMap := map[string]any{
		"username":  ciCfg.Username,
		"password":  ciCfg.Password,
		"sshPubKey": ciCfg.SSHPubKey,
		"mac":       ciCfg.MAC,
		"cidr":      ciCfg.CIDR,
		"mtu":       ciCfg.MTU,
		"ifname":    ciCfg.IFName,
		"defaultGW": map[string]any{
			"ip":     ciCfg.DefaultGW.IP,
			"onLink": ciCfg.DefaultGW.OnLink,
		},
		"commands": ciCfg.Commands,
		"files":    []map[string]any{},

		"instanceID": ciCfg.InstanceID,
		"hostname":   ciCfg.Hostname,
		"osType":     ciCfg.OS.Type,
	}
	for k, v := range ciCfg.Files {
		dataMap["files"] = append(dataMap["files"].([]map[string]any), map[string]any{
			"path":    k,
			"content": base64.StdEncoding.EncodeToString(v),
		})
	}
	var udataTmplFile, mdataTmplFile, networkTmplFile string
	var userData, metaData, networkData string
	switch ciCfg.OS.Type {
	case "windows":
		udataTmplFile = filepath.Join(configs.Conf.VirtTmplDir, "windows", "user-data.yaml")
		mdataTmplFile = filepath.Join(configs.Conf.VirtTmplDir, "windows", "meta-data.json")
		networkTmplFile = filepath.Join(configs.Conf.VirtTmplDir, "windows", "network-config.yaml")
		userData = winUserData
		metaData = winMetaData
		networkData = winNetworkData
	default:
		udataTmplFile = filepath.Join(configs.Conf.VirtTmplDir, "user-data.yaml")
		mdataTmplFile = filepath.Join(configs.Conf.VirtTmplDir, "meta-data.yaml")
		networkTmplFile = filepath.Join(configs.Conf.VirtTmplDir, "network-config.yaml")
		userData = linuxUserData
		metaData = linuxMetaData
		networkData = linuxNetworkData
	}

	uDataBS, err := template.Render(udataTmplFile, userData, dataMap)
	if err != nil {
		return "", "", "", err
	}

	mDataBS, err := template.Render(mdataTmplFile, metaData, dataMap)
	if err != nil {
		return "", "", "", err
	}
	networkBS, err := template.Render(networkTmplFile, networkData, dataMap)
	if err != nil {
		return "", "", "", err
	}
	return string(uDataBS), string(mDataBS), string(networkBS), nil
}

func (ciCfg *CloudInitConfig) GenerateISO(fname string) (err error) {
	dir, err := os.MkdirTemp("/tmp", "cloud-init")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)
	udataFname := filepath.Join(dir, "user-data")
	mdataFname := filepath.Join(dir, "meta-data")
	networkFname := filepath.Join(dir, "network-config")

	udata, mdata, ndata, err := ciCfg.GenFilesContent()
	if err != nil {
		return
	}
	if err := os.WriteFile(udataFname, []byte(udata), 0600); err != nil {
		return errors.Wrap(err, "")
	}
	if err := os.WriteFile(mdataFname, []byte(mdata), 0600); err != nil {
		return errors.Wrap(err, "")
	}

	if err := os.WriteFile(networkFname, []byte(ndata), 0600); err != nil {
		return errors.Wrap(err, "")
	}
	// args := []string{
	// 	"genisoimage", "-output", fname, "-V", "cidata", "-r", "-J", "user-data", "meta-data",
	// }
	args := []string{
		"cloud-localds", "--network-config=network-config", fname, "user-data", "meta-data",
	}
	cmd := exec.Command(args[0], args[1:]...) //nolint
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to exec genisoimage %s", out)
	}
	return
}

func extractISO(isoPath, outputDir string) error {
	isoFile, err := os.Open(isoPath)
	if err != nil {
		return err
	}
	defer isoFile.Close()

	return util.ExtractImageToDirectory(isoFile, outputDir)
}

func (ciCfg *CloudInitConfig) ReplaceUserData(fname string) (err error) {
	if ciCfg.Username == "" || ciCfg.Password == "" {
		return
	}
	// if the iso file doesn't exist, it means the cloud-init DataSource is not NoCloud-local
	_, err = os.Stat(fname)
	if os.IsNotExist(err) {
		return nil
	}

	dir, err := os.MkdirTemp("/tmp", "cloud-init")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)
	if err := extractISO(fname, dir); err != nil {
		return err
	}
	udataFname := filepath.Join(dir, "user-data")

	udata, _, _, err := ciCfg.GenFilesContent()
	if err := os.WriteFile(udataFname, []byte(udata), 0600); err != nil {
		return errors.Wrap(err, "")
	}
	args := []string{
		"cloud-localds", "--network-config=network-config", fname, "user-data", "meta-data",
	}
	cmd := exec.Command(args[0], args[1:]...) //nolint
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to exec genisoimage %s", out)
	}
	return
}
