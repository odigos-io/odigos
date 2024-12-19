package migrations

type Migration interface {
	Name() string           // Unique name of the migration
	Description() string    // A brief description of the migration
	TriggerVersion() string // The version at which the migration becomes applicable
	Execute() error         // Code to execute the migration
}
