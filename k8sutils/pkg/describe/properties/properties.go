package properties

type PropertyStatus string

const (

	// the property is in it's desired state
	PropertyStatusSuccess PropertyStatus = "success"

	// the property is not in it's desired state, but it's state might be temporary
	// if wait some time, it might reconcile to the desired state (or not)
	PropertyStatusTransitioning PropertyStatus = "transitioning"

	// the property is not in it's desired state, and it's state is not expected to change
	PropertyStatusError PropertyStatus = "error"
)

type EntityProperty struct {

	// The name of the property being described
	Name string `json:"name"`

	// Explain is a human readable explanation of the property
	// it is static describe the property itself and not the current value.
	// this is optional and can be empty
	Explain string `json:"explain,omitempty"`

	// The value to display for this property
	Value interface{} `json:"value"`

	// The status of the property actual state
	Status PropertyStatus `json:"status,omitempty"`
}
