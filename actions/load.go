package actions

import (
	"embed"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/*
var actionsFS embed.FS

// array of all action catalog configs
var loadedActions []Action

// map from action type to action catalog config object
var actionsByType map[string]Action

func Load() error {
	return load(actionsFS)
}

func Get() []Action {
	return loadedActions
}

func GetActionByType(actionType string) (Action, bool) {
	action, ok := actionsByType[actionType]
	return action, ok
}

// fsys is an interface rather than the concrete embed.FS so the catalog can be loaded from a
// synthetic filesystem in tests; Load always passes the embedded one.
func load(fsys fs.ReadDirFS) error {
	var acts []Action
	var actsByTypeMap = make(map[string]Action)

	files, err := fsys.ReadDir("data")
	if err != nil {
		return err
	}

	for _, file := range files {
		fileName := file.Name()

		if !strings.HasSuffix(fileName, ".yaml") && !strings.HasSuffix(fileName, ".yml") {
			continue
		}

		bytesData, err := fs.ReadFile(fsys, "data/"+file.Name())
		if err != nil {
			return err
		}

		var act Action
		err = yaml.Unmarshal(bytesData, &act)
		if err != nil {
			return err
		}

		actsByTypeMap[act.Metadata.Type] = act
		acts = append(acts, act)
	}

	actionsByType = actsByTypeMap
	loadedActions = acts
	return nil
}
