package state

import (
	"slices"

	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

// moveShapes moves selectedShapes to the positions newPosition returns, and the lines
// connected to them. newPosition returns false to keep a shape.
func (m *Manager) moveShapes(selectedShapes []shape.Shape, isUpdateConfirmed bool, newPosition func(shape.Shape) (geo.Point, bool)) {
	shapeConnector := m.shapeManager.ShapeConnector()

	var selectedLines []*shape.Line
	var selectedNonLineShapes []shape.Shape
	for _, selected := range selectedShapes {
		if line, ok := selected.(*shape.Line); ok {
			selectedLines = append(selectedLines, line)
		} else {
			selectedNonLineShapes = append(selectedNonLineShapes, selected)
		}
	}

	// lineIDToHeadCount is how many heads of each line connect to the selected shapes.
	lineIDToHeadCount := map[string]int{}
	var connectedLineIDs []string
	for _, selected := range selectedNonLineShapes {
		for _, connector := range shapeConnector.Connectors(selected) {
			if lineIDToHeadCount[connector.LineID] == 0 {
				connectedLineIDs = append(connectedLineIDs, connector.LineID)
			}
			lineIDToHeadCount[connector.LineID]++
		}
	}
	lineIDToConnectionCount := map[string]int{}
	for _, line := range selectedLines {
		lineIDToConnectionCount[line.ID()] = m.connectionCount(line)
	}

	updatePosition := func(target shape.Shape) (geo.Rect, bool) {
		position, ok := newPosition(target)
		if !ok {
			return geo.Rect{}, false
		}
		newBound := geo.Rect{Position: position, Size: target.Bound().Size}
		m.shapeManager.Execute(shape.ChangeBound{Target: target, NewBound: newBound})
		return newBound, true
	}

	affectedLineIDs := map[string]bool{}
	for _, target := range selectedNonLineShapes {
		newBound, ok := updatePosition(target)
		if !ok {
			continue
		}
		for _, connector := range shapeConnector.Connectors(target) {
			// Only move the line end if the line has one head on the selected shapes, and the
			// line is not selected or is selected with both heads connected.
			// TODO: This logic is too complicated (from the original).
			connectionCount, ok := lineIDToConnectionCount[connector.LineID]
			if !ok {
				connectionCount = 2
			}
			if lineIDToHeadCount[connector.LineID] == 1 && connectionCount == 2 {
				affectedLineIDs[connector.LineID] = true
				m.updateConnector(connector, newBound, isUpdateConfirmed)
			}
		}
	}

	var lineIDs []string
	for _, lineID := range connectedLineIDs {
		if lineIDToHeadCount[lineID] == 2 {
			lineIDs = append(lineIDs, lineID)
		}
	}
	for _, line := range selectedLines {
		if !affectedLineIDs[line.ID()] && !slices.Contains(lineIDs, line.ID()) {
			lineIDs = append(lineIDs, line.ID())
		}
	}

	for _, lineID := range lineIDs {
		found, ok := m.shapeManager.Shape(lineID)
		if !ok {
			continue
		}
		line, ok := found.(*shape.Line)
		if !ok {
			continue
		}
		oldBound := line.Bound()
		newBound, ok := updatePosition(line)
		if !ok || newBound == oldBound {
			continue
		}
		// The line moves alone, so it is not connected anymore.
		if _, ok := lineIDToHeadCount[lineID]; !ok {
			shapeConnector.RemoveConnector(line, shape.AnchorStart)
			shapeConnector.RemoveConnector(line, shape.AnchorEnd)
		}
	}
}

// updateConnectors moves the ends of the lines connected to target.
func (m *Manager) updateConnectors(target shape.Shape, newBound geo.Rect, isUpdateConfirmed bool) {
	for _, connector := range m.shapeManager.ShapeConnector().Connectors(target) {
		m.updateConnector(connector, newBound, isUpdateConfirmed)
	}
}

func (m *Manager) updateConnector(connector shape.LineConnector, newBound geo.Rect, isUpdateConfirmed bool) {
	found, ok := m.shapeManager.Shape(connector.LineID)
	if !ok {
		return
	}
	line, ok := found.(*shape.Line)
	if !ok {
		return
	}
	anchorPointUpdate := shape.AnchorPointUpdate{
		Anchor: connector.Anchor,
		Point:  connector.PointInNewBound(line.Direction(connector.Anchor), newBound),
	}
	// Not through the MoveLineAnchor command, which removes the connector when it has no shape
	// to connect. The shape which moves updates the versions up to the root.
	line.MoveAnchorPoint(anchorPointUpdate, isUpdateConfirmed, true)
}

func (m *Manager) connectionCount(line *shape.Line) int {
	count := 0
	for _, anchor := range []shape.Anchor{shape.AnchorStart, shape.AnchorEnd} {
		if m.shapeManager.ShapeConnector().HasConnector(line, anchor) {
			count++
		}
	}
	return count
}
