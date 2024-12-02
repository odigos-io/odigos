package size

import (
	"reflect"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/common"
)

const (
	SizeS common.ProfileName = "size_s"
	SizeM common.ProfileName = "size_m"
	SizeL common.ProfileName = "size_l"
)

func GetGatewayConfigBasedOnSize(profile common.ProfileName) *common.CollectorGatewayConfiguration {

	AggregateProfiles := append([]common.ProfileName{profile}, resources.ProfilesMap[profile].Dependencies...)

	for _, profile := range AggregateProfiles {
		switch profile {
		case SizeS:
			return &common.CollectorGatewayConfiguration{
				MinReplicas:      1,
				MaxReplicas:      5,
				RequestCPUm:      150,
				LimitCPUm:        300,
				RequestMemoryMiB: 300,
			}
		case SizeM:
			return &common.CollectorGatewayConfiguration{
				MinReplicas:      2,
				MaxReplicas:      8,
				RequestCPUm:      500,
				LimitCPUm:        1000,
				RequestMemoryMiB: 500,
			}
		case SizeL:
			return &common.CollectorGatewayConfiguration{
				MinReplicas:      3,
				MaxReplicas:      12,
				RequestCPUm:      750,
				LimitCPUm:        1250,
				RequestMemoryMiB: 750,
			}
		}
	}
	// Return nil if no matching profile is found.
	return nil
}

func RemoveSizingProfileAddedValues(target, source *common.CollectorGatewayConfiguration) {
	if target == nil || source == nil {
		return
	}

	// Use reflection to iterate over the fields of CollectorGatewayConfiguration.
	tVal := reflect.ValueOf(target).Elem()
	sVal := reflect.ValueOf(source).Elem()

	tType := tVal.Type()
	for i := 0; i < tVal.NumField(); i++ {
		tField := tVal.Field(i)
		sField := sVal.Field(i)

		// Compare fields, and if they match, set the target field to its zero value.
		if tField.CanSet() && reflect.DeepEqual(tField.Interface(), sField.Interface()) {
			tField.Set(reflect.Zero(tType.Field(i).Type))
		}
	}
}

func FilterSizeProfiles(profiles []common.ProfileName) common.ProfileName {
	for _, profile := range profiles {

		// Check if the profile is a size profile.
		switch profile {
		case SizeS, SizeM, SizeL:
			return profile
		}

		// Check if the profile has a dependency which is a size profile.
		profileDependencies := resources.ProfilesMap[profile].Dependencies
		for _, dependencyProfile := range profileDependencies {
			switch dependencyProfile {
			case SizeS, SizeM, SizeL:
				return dependencyProfile
			}
		}
	}
	return ""
}
