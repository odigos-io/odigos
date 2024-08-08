package envs

const NodeVersionConst = "NODE_VERSION"
const PythonVersionConst = "PYTHON_VERSION"

// EnvValuesMap is a map of environment variables and their separators
var EnvValuesMap = map[string]string{
	NodeVersionConst:   " ",
	PythonVersionConst: ":",
}
