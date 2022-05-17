package pie

const (
	KSize = 24
	Alpha = 3
)

const (
	IDLen             = KSize
	UserCertHashLen   = 20
	ServerCertHashLen = 50
)

const (
	MetaDataRedundancy = 12
	FileDataRedundancy = 4
)

const (
	MaxMessageLen = 2 * 1024 * 1024
)

const (
	UserTLSProto    = "pie-q-u-1"
	TrackerTLSProto = "pie-q-t-1"
)
