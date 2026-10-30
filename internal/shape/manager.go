package shape

import "github.com/bluemir/monosketch/internal/observable"

// Manager holds all shapes of a sketch and changes them with commands.
type Manager struct {
	root           *Group
	allShapes      map[string]Shape
	shapeConnector *ShapeConnector

	// RootValue is notified when the root is replaced or the document info changes.
	RootValue *observable.Value[*Group]
	// Version is the version of the root. Observe it to know when shapes change.
	Version *observable.Value[int]
}

func NewManager() *Manager {
	root := NewGroup("", "")
	manager := &Manager{
		root:      root,
		RootValue: observable.New(root),
		Version:   observable.New(root.versionCode),
	}
	manager.ReplaceRoot(root, NewShapeConnector())
	return manager
}

func (m *Manager) Root() *Group                    { return m.root }
func (m *Manager) ShapeConnector() *ShapeConnector { return m.shapeConnector }

// ReplaceRoot replaces all shapes with the shapes in newRoot.
func (m *Manager) ReplaceRoot(newRoot *Group, newConnector *ShapeConnector) {
	currentVersion := m.root.versionCode
	m.root = newRoot
	m.RootValue.Set(newRoot)

	m.shapeConnector = newConnector

	m.allShapes = map[string]Shape{newRoot.id: newRoot}
	registerAll(newRoot, m.allShapes)

	if currentVersion == newRoot.versionCode {
		m.Version.Set(currentVersion - 1)
	} else {
		m.Version.Set(newRoot.versionCode)
	}
}

// registerAll adds every shape under group into shapes.
func registerAll(group *Group, shapes map[string]Shape) {
	for _, shape := range group.Items() {
		shapes[shape.ID()] = shape
		if child, ok := shape.(*Group); ok {
			// The original recurses into group itself here, which never ends when there is a
			// child group.
			registerAll(child, shapes)
		}
	}
}

func (m *Manager) Execute(command Command) {
	parent := command.affectedParent(m)
	if parent == nil {
		return
	}
	ancestors := m.ancestors(parent)
	currentVersion := parent.versionCode

	command.execute(m, parent)

	if _, ok := m.allShapes[parent.id]; currentVersion == parent.versionCode && ok {
		return
	}
	for _, ancestor := range ancestors {
		ancestor.update(func() bool { return true })
	}
	m.Version.Set(m.root.versionCode)
}

// group returns the root if shapeID is empty, and nil if shapeID is not a group.
func (m *Manager) group(shapeID string) *Group {
	if shapeID == "" {
		return m.root
	}
	group, _ := m.allShapes[shapeID].(*Group)
	return group
}

func (m *Manager) Shape(shapeID string) (Shape, bool) {
	shape, ok := m.allShapes[shapeID]
	return shape, ok
}

func (m *Manager) register(shape Shape) {
	m.allShapes[shape.ID()] = shape
}

func (m *Manager) unregister(shape Shape) {
	delete(m.allShapes, shape.ID())
}

func (m *Manager) ancestors(group *Group) []*Group {
	var result []*Group
	parent, _ := m.allShapes[group.parentID].(*Group)
	for parent != nil {
		result = append(result, parent)
		parent, _ = m.allShapes[parent.parentID].(*Group)
	}
	return result
}

// NotifyDocumentUpdate tells that the document changed, including info like its name, not only
// the shapes.
func (m *Manager) NotifyDocumentUpdate() {
	m.RootValue.Set(m.root)
}
