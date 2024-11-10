package properties

func GetTextCreated(created bool) string {
	if created {
		return "created"
	} else {
		return "not created"
	}
}

func GetSuccessOrTransitioning(matchExpected bool) PropertyStatus {
	if matchExpected {
		return PropertyStatusSuccess
	} else {
		return PropertyStatusTransitioning
	}
}

func GetSuccessOrError(matchExpected bool) PropertyStatus {
	if matchExpected {
		return PropertyStatusSuccess
	} else {
		return PropertyStatusError
	}
}
