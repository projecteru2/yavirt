package models

const (
	// HostVirtType .
	HostVirtType = "virt"
	// HostMetaType .
	HostMetaType = "meta"

	// LabelPublish .
	LabelPublish = "Publish"
	// LabelHealthCheck .
	LabelHealthCheck = "HealthCheck"

	// MaxMaskBits at most /30
	MaxMaskBits = 30
	// MinMaskBits at least /1
	MinMaskBits = 1
	// MaxMaskBitsForBlocks at most /23
	MaxMaskBitsForBlocks = 23
	// BlockFlagSpawned .
	BlockFlagSpawned = 1
	// BlockFlagUnspawned .
	BlockFlagUnspawned = 0
	// IPFlagFree .
	IPFlagFree = 0
	// IPFlagAssigned .
	IPFlagAssigned = 1
	// MaxBlockIPCount .
	MaxBlockIPCount = 256
)
