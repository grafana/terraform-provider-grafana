package k6providerapi

type K6APIConfig struct {
	// Token is the k6 Cloud API token.
	Token string

	// StackID is the k6 Cloud API stack id.
	StackID int32
}
