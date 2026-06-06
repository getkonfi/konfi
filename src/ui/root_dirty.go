package ui

import "github.com/eminert/konfi/pkg"

type dirtyConfigState struct {
	config     *pkg.ConfigFile
	origValues map[string]string
	undoStack  *UndoStack
	fileState  string
}

func cloneValues(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func (r *root) stashActiveDirtyConfig() (string, bool) {
	if r.content.konfable == nil || r.content.config == nil {
		return "", false
	}
	name := r.content.konfable.Name()
	if r.dirtyConfigs == nil {
		r.dirtyConfigs = make(map[string]dirtyConfigState)
	}
	if !r.content.config.Dirty() {
		delete(r.dirtyConfigs, name)
		return name, false
	}
	state := dirtyConfigState{
		config:     r.content.config,
		origValues: cloneValues(r.content.origValues),
		fileState:  r.content.fileState,
	}
	if r.content.undoStack != nil {
		state.undoStack = r.content.undoStack.Clone()
	}
	r.dirtyConfigs[name] = state
	return name, true
}

func (r *root) takeDirtyConfig(name, path string) (dirtyConfigState, bool) {
	if r.dirtyConfigs == nil {
		return dirtyConfigState{}, false
	}
	state, ok := r.dirtyConfigs[name]
	if !ok {
		return dirtyConfigState{}, false
	}
	if state.config == nil || !state.config.Dirty() {
		delete(r.dirtyConfigs, name)
		return dirtyConfigState{}, false
	}
	delete(r.dirtyConfigs, name)
	if path != "" {
		state.config.Path = path
	}
	return state, true
}

func (r *root) clearDirtyConfig(name string) {
	if r.dirtyConfigs == nil || name == "" {
		return
	}
	delete(r.dirtyConfigs, name)
}
