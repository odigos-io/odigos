package category

func GetPercentageOrDefault(percentage *float64, defaultValue float64) float64 {
	if percentage == nil {
		return defaultValue
	}
	return *percentage
}

func GetPercentageOrDefault100(percentage *float64) float64 {
	return GetPercentageOrDefault(percentage, 100.0)
}
