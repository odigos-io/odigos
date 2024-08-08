package envs

const NodeVersionConst = "NODE_VERSION"
const PythonVersionConst = "PYTHON_VERSION"

// EnvDetailsSeparatorMap is a map of environment variables and their separators
var EnvDetailsSeparatorMap = map[string]string{
	NodeVersionConst:   " ",
	PythonVersionConst: ":",
}
