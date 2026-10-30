package state

import "github.com/bluemir/monosketch/internal/shape"

// Backup is the state of shapes to keep in the history and the local storage.
type Backup struct {
	Root       *shape.SerializableGroup
	Connectors []shape.SerializableLineConnector
}

// historyStack keeps the backups for undo and redo. The last item of undoStack is the
// current state.
type historyStack struct {
	undoStack []historyItem
	redoStack []historyItem
}

type historyItem struct {
	versionCode int
	backup      Backup
}

func (s *historyStack) push(versionCode int, backup Backup) {
	if len(s.undoStack) > 0 && s.undoStack[len(s.undoStack)-1].versionCode == versionCode {
		return
	}
	s.undoStack = append(s.undoStack, historyItem{versionCode: versionCode, backup: backup})
	s.redoStack = nil
}

func (s *historyStack) clear() {
	s.undoStack = nil
	s.redoStack = nil
}

func (s *historyStack) undo() (historyItem, bool) {
	if len(s.undoStack) <= 1 {
		return historyItem{}, false
	}
	current := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.redoStack = append(s.redoStack, current)
	return s.undoStack[len(s.undoStack)-1], true
}

func (s *historyStack) redo() (historyItem, bool) {
	if len(s.redoStack) == 0 {
		return historyItem{}, false
	}
	current := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]
	s.undoStack = append(s.undoStack, current)
	return current, true
}

// Restore replaces the shapes with backup, or with an empty document of rootID if backup is
// nil. JS calls this on start with the document read from the local storage.
func (m *Manager) Restore(rootID string, backup *Backup) {
	if backup == nil {
		m.replaceRoot(shape.NewGroup(rootID, ""), shape.NewShapeConnector())
		return
	}
	m.replaceRoot(shape.RestoreRoot(backup.Root), shape.RestoreShapeConnector(backup.Connectors))
}

func (m *Manager) replaceRoot(newRoot *shape.Group, newConnector *shape.ShapeConnector) {
	if m.shapeManager.Root().ID() != newRoot.ID() {
		m.history.clear()
	}
	m.shapeManager.ReplaceRoot(newRoot, newConnector)
	m.workingParentGroup = m.shapeManager.Root()
	m.selection.Clear()
}

// requestBackupIfIdle asks JS for a backup when shapes are changed outside editing. See
// ADR-0007.
func (m *Manager) requestBackupIfIdle() {
	editingMode := m.EditingMode.Value()
	versionCode := m.shapeManager.Version.Value()
	if editingMode.IsEditing || (editingMode.SkippedVersion != nil && *editingMode.SkippedVersion == versionCode) {
		return
	}
	m.backupRequestedVersion = &versionCode
	m.notify(ChangeBackup)
}

// Backup keeps the current state in the history and returns it for the local storage. JS
// calls this some time after ChangeBackup. It returns false if shapes changed after the
// request, since another request follows.
func (m *Manager) Backup() (Backup, bool) {
	requested := m.backupRequestedVersion
	if requested == nil || *requested != m.shapeManager.Version.Value() {
		return Backup{}, false
	}
	m.backupRequestedVersion = nil

	root := m.shapeManager.Root()
	backup := Backup{
		Root:       root.Serializable(true).(*shape.SerializableGroup),
		Connectors: m.shapeManager.ShapeConnector().Serializable(),
	}
	m.history.push(root.VersionCode(), backup)
	return backup, true
}

func (m *Manager) undo() {
	if item, ok := m.history.undo(); ok {
		m.applyHistory(item)
	}
}

func (m *Manager) redo() {
	if item, ok := m.history.redo(); ok {
		m.applyHistory(item)
	}
}

func (m *Manager) applyHistory(item historyItem) {
	m.replaceRoot(shape.RestoreRoot(item.backup.Root), shape.RestoreShapeConnector(item.backup.Connectors))
}
