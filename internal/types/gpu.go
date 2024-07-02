package types

type GPUInfo struct {
	Address string `json:"address" mapstructure:"address"`
	Index   int    `json:"index" mapstructure:"index"`
	// example value: "NVIDIA Corporation"
	Vendor string `json:"vendor" mapstructure:"vendor"`
	// example value: "GA104 [GeForce RTX 3070]"
	Product string `json:"product" mapstructure:"product"`

	// NUMA NUMAInfo
	NumaID string `json:"numa_id" mapstructure:"numa_id"`

	// Cores   int   `json:"cores" mapstructure:"cores"`
	GMemory int64 `json:"gmemory" mapstructure:"gmemory"`
}

type GPUEngineParams struct {
	Addrs []string `json:"addrs" mapstructure:"addrs"`
}
