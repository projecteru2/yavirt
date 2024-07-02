package types

import (
	resourcetypes "github.com/projecteru2/core/resource/types"
)

// Node .
type Node struct {
	Name      string
	Endpoint  string
	Podname   string
	Labels    map[string]string
	Available bool
}

// NodeStatus .
type NodeStatus struct {
	Nodename string
	Podname  string
	Alive    bool
	Error    error
}

type NodeResource struct {
	Capacity resourcetypes.Resources
}

type Workload struct {
	ID string
}

type SetNodeOpts struct {
	Nodename      string
	Endpoint      string
	Ca            string
	Cert          string
	Key           string
	Labels        map[string]string
	Resources     map[string][]byte
	Delta         bool
	WorkloadsDown bool
}

type AddNodeOpts struct {
	Nodename  string
	Endpoint  string
	Podname   string
	Ca        string
	Cert      string
	Key       string
	Labels    map[string]string
	Resources map[string][]byte
}
