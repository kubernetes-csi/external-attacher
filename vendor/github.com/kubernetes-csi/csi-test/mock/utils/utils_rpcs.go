package utils

const (
	// Namespace is the namesapce used by the protobuf.
	Namespace = "csi"

	// CSIEndpoint is the name of the environment variable that
	// contains the CSI endpoint.
	CSIEndpoint = "CSI_ENDPOINT"

	//
	// Controller Service
	//
	ctrlSvc = "/" + Namespace + ".Controller/"

	// CreateVolume is the full method name for the
	// eponymous RPC message.
	CreateVolume = ctrlSvc + "CreateVolume"

	// DeleteVolume is the full method name for the
	// eponymous RPC message.
	DeleteVolume = ctrlSvc + "DeleteVolume"

	// ControllerPublishVolume is the full method name for the
	// eponymous RPC message.
	ControllerPublishVolume = ctrlSvc + "ControllerPublishVolume"

	// ControllerUnpublishVolume is the full method name for the
	// eponymous RPC message.
	ControllerUnpublishVolume = ctrlSvc + "ControllerUnpublishVolume"

	// ValidateVolumeCapabilities is the full method name for the
	// eponymous RPC message.
	ValidateVolumeCapabilities = ctrlSvc + "ValidateVolumeCapabilities"

	// ListVolumes is the full method name for the
	// eponymous RPC message.
	ListVolumes = ctrlSvc + "ListVolumes"

	// GetCapacity is the full method name for the
	// eponymous RPC message.
	GetCapacity = ctrlSvc + "GetCapacity"

	// ControllerGetCapabilities is the full method name for the
	// eponymous RPC message.
	ControllerGetCapabilities = ctrlSvc + "ControllerGetCapabilities"

	// ControllerProbe is the full method name for the
	// eponymous RPC message.
	ControllerProbe = ctrlSvc + "ControllerProbe"

	//
	// Identity Service
	//
	identSvc = "/" + Namespace + ".Identity/"

	// GetSupportedVersions is the full method name for the
	// eponymous RPC message.
	GetSupportedVersions = identSvc + "GetSupportedVersions"

	// GetPluginInfo is the full method name for the
	// eponymous RPC message.
	GetPluginInfo = identSvc + "GetPluginInfo"

	//
	// Node Service
	//
	nodeSvc = "/" + Namespace + ".Node/"

	// NodeGetId is the full method name for the
	// eponymous RPC message.
	NodeGetId = nodeSvc + "NodeGetId"

	// NodePublishVolume is the full method name for the
	// eponymous RPC message.
	NodePublishVolume = nodeSvc + "NodePublishVolume"

	// NodeUnpublishVolume is the full method name for the
	// eponymous RPC message.
	NodeUnpublishVolume = nodeSvc + "NodeUnpublishVolume"

	// NodeProbe is the full method name for the
	// eponymous RPC message.
	NodeProbe = nodeSvc + "NodeProbe"

	// NodeGetCapabilities is the full method name for the
	// eponymous RPC message.
	NodeGetCapabilities = nodeSvc + "NodeGetCapabilities"
)
