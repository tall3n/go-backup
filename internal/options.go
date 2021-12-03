package internal

type CommandLineArgs struct {
	// AWS Region
	Region string
	// AWS Access Key ID
	AccessKeyID string
	// AWS Secret Access Key
	SecretAccessKey string
	// Filter
	Filter string
	//Prefix - Unique prefix for all resources
	Prefix string

	ResourceTypes []string

	DryRun bool
}
