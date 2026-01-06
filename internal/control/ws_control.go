// Package control handles input protocol and gesture mapping.
package control

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
	"github.com/frudas24/deskslice/internal/session"
	"github.com/frudas24/deskslice/internal/wininput"
	"github.com/gorilla/websocket"
)

// MonitorProvider returns the current list of monitors.
type MonitorProvider func() ([]monitor.Monitor, error)

// Server handles websocket control input.
type Server struct {
	mu               sync.Mutex
	upgrader         websocket.Upgrader
	session          *session.Session
	injector         wininput.Injector
	gestures         *GestureState
	listMonitors     MonitorProvider
	onPipelineChange func(reason string)
	saveCalib        func(calib.Calib) error
	conn             *websocket.Conn
}

// NewServer creates a control websocket server.
func NewServer(sess *session.Session, injector wininput.Injector, listMonitors MonitorProvider, onPipelineChange func(reason string), saveCalib func(calib.Calib) error) *Server {
	return &Server{
		session:      sess,
		injector:     injector,
		listMonitors: listMonitors,
		gestures:     NewGestureState(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
		onPipelineChange: onPipelineChange,
		saveCalib:        saveCalib,
	}
}

// ServeHTTP upgrades the connection and processes control messages.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.session.IsAuthenticated() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	if err := s.acceptConn(conn); err != nil {
		_ = conn.Close()
		return
	}
	defer s.cleanupConn(conn)

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if err := s.handleMessage(msg); err != nil {
			return
		}
	}
}

// acceptConn ensures only one active control connection exists.
func (s *Server) acceptConn(conn *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		return fmt.Errorf("control connection already active")
	}
	s.conn = conn
	return nil
}

// cleanupConn clears the active connection when closed.
func (s *Server) cleanupConn(conn *websocket.Conn) {
	s.mu.Lock()
	if s.conn == conn {
		s.conn = nil
	}
	s.mu.Unlock()
	_ = conn.Close()
}

// handleMessage dispatches a single control message.
func (s *Server) handleMessage(msg Message) error {
	switch msg.T {
	case "down":
		return s.handlePointerDown(msg)
	case "move":
		return s.handlePointerMove(msg)
	case "up":
		return s.handlePointerUp(msg)
	case "type":
		return s.handleType(msg.Text)
	case "enter":
		return s.handleEnter()
	case "clearChat":
		return s.handleClearChat()
	case "setMode":
		s.session.SetMode(msg.Mode)
		s.notifyPipeline("mode")
		return nil
	case "setMonitor":
		s.session.SetMonitor(msg.Idx)
		s.notifyPipeline("monitor")
		return nil
	case "restartPresetup":
		s.session.SetMode(session.ModePresetup)
		s.notifyPipeline("restart_presetup")
		return nil
	case "setVideo":
		s.session.SetVideoMode(msg.Video)
		s.notifyPipeline("video")
		return nil
	case "calibRect":
		return s.handleCalibRect(msg)
	case "inputEnabled":
		if msg.Enabled != nil {
			s.session.SetInputEnabled(*msg.Enabled)
		}
		return nil
	default:
		return nil
	}
}

// handlePointerDown handles pointer down events.
func (s *Server) handlePointerDown(msg Message) error {
	c := s.session.GetCalib()
	absX, absY, mode, pluginAbs, err := s.mapCoordsWithCalib(msg.X, msg.Y, c)
	if err != nil {
		return err
	}
	if mode == session.ModeRun {
		actions := s.gestures.HandleDown(s.session.InputEnabled(), msg.ID, absX, absY, pluginAbs, c.ScrollRel)
		return s.applyActions(actions)
	}
	actions := buildPresetupActions("down", absX, absY)
	return s.applyActions(actions)
}

// handlePointerMove handles pointer move events.
func (s *Server) handlePointerMove(msg Message) error {
	c := s.session.GetCalib()
	absX, absY, mode, _, err := s.mapCoordsWithCalib(msg.X, msg.Y, c)
	if err != nil {
		return err
	}
	if mode == session.ModeRun {
		actions := s.gestures.HandleMove(s.session.InputEnabled(), msg.ID, absX, absY)
		return s.applyActions(actions)
	}
	actions := buildPresetupActions("move", absX, absY)
	return s.applyActions(actions)
}

// handlePointerUp handles pointer up events.
func (s *Server) handlePointerUp(msg Message) error {
	c := s.session.GetCalib()
	absX, absY, mode, _, err := s.mapCoordsWithCalib(msg.X, msg.Y, c)
	if err != nil {
		return err
	}
	if mode == session.ModeRun {
		actions := s.gestures.HandleUp(s.session.InputEnabled(), msg.ID, absX, absY)
		return s.applyActions(actions)
	}
	actions := buildPresetupActions("up", absX, absY)
	return s.applyActions(actions)
}

// handleType handles type messages.
func (s *Server) handleType(text string) error {
	c := s.session.GetCalib()
	pluginAbs, err := s.pluginAbsVirtual(c)
	if err != nil {
		return err
	}
	chatAbs := chatRectAbsFromPlugin(pluginAbs, c.ChatRel)
	actions := ActionsForType(s.session.InputEnabled(), text, chatAbs)
	return s.applyActions(actions)
}

// handleEnter handles enter messages.
func (s *Server) handleEnter() error {
	c := s.session.GetCalib()
	pluginAbs, err := s.pluginAbsVirtual(c)
	if err != nil {
		return err
	}
	chatAbs := chatRectAbsFromPlugin(pluginAbs, c.ChatRel)
	actions := ActionsForEnter(s.session.InputEnabled(), chatAbs)
	return s.applyActions(actions)
}

// handleClearChat focuses the chat input and clears its contents.
func (s *Server) handleClearChat() error {
	if !s.session.InputEnabled() {
		return nil
	}
	c := s.session.GetCalib()
	pluginAbs, err := s.pluginAbsVirtual(c)
	if err != nil {
		return err
	}
	chat := calib.Normalize(c.ChatRel)
	if chat.W <= 0 || chat.H <= 0 {
		return fmt.Errorf("chat rect not calibrated")
	}
	chatAbs := chatRectAbsFromPlugin(pluginAbs, c.ChatRel)
	x, y := centerPoint(chatAbs)
	if err := s.injector.ClickAt(x, y); err != nil {
		return err
	}
	if err := s.injector.SelectAll(); err != nil {
		return err
	}
	return s.injector.Delete()
}

// handleCalibRect updates calibration state.
func (s *Server) handleCalibRect(msg Message) error {
	if msg.Rect == nil {
		return nil
	}
	c := s.session.GetCalib()
	rect := calib.Rect{X: msg.Rect.X, Y: msg.Rect.Y, W: msg.Rect.W, H: msg.Rect.H}

	switch msg.Step {
	case "plugin":
		c.PluginAbs = rect
		c.MonitorIndex = s.session.Monitor()
		s.notifyPipeline("plugin_rect")
	case "chat":
		c.ChatRel = rect
	case "scroll":
		c.ScrollRel = rect
	default:
		return nil
	}

	s.session.SetCalib(c)
	if s.saveCalib != nil {
		if err := s.saveCalib(c); err != nil {
			return err
		}
	}
	return nil
}

// mapCoordsWithCalib converts normalized coords into absolute screen coordinates using a consistent calibration snapshot.
func (s *Server) mapCoordsWithCalib(xn, yn float64, c calib.Calib) (int, int, string, calib.Rect, error) {
	mode := s.session.Mode()
	if mode == session.ModeRun {
		pluginAbs, err := s.pluginAbsVirtual(c)
		if err != nil {
			return 0, 0, mode, calib.Rect{}, err
		}
		x, y := NormToAbsRun(xn, yn, pluginAbs)
		return x, y, mode, pluginAbs, nil
	}

	monitors, err := s.listMonitors()
	if err != nil {
		return 0, 0, mode, calib.Rect{}, err
	}
	monitorIndex := s.session.Monitor()
	m, ok := monitor.GetMonitorByIndex(monitors, monitorIndex)
	if !ok {
		return 0, 0, mode, calib.Rect{}, fmt.Errorf("monitor %d not found", monitorIndex)
	}
	x, y := NormToAbsPresetup(xn, yn, m)
	return x, y, mode, calib.Rect{}, nil
}

// applyActions executes actions using the injector.
func (s *Server) applyActions(actions []Action) error {
	for _, action := range actions {
		if err := s.applyAction(action); err != nil {
			return err
		}
	}
	return nil
}

// applyAction executes a single action.
func (s *Server) applyAction(action Action) error {
	switch action.Type {
	case ActMove:
		return s.injector.MoveAbs(action.X, action.Y)
	case ActLeftDown:
		return s.injector.LeftDown()
	case ActLeftUp:
		return s.injector.LeftUp()
	case ActClick:
		return s.injector.ClickAt(action.X, action.Y)
	case ActType:
		return s.injector.TypeUnicode(action.Text)
	case ActEnter:
		return s.injector.Enter()
	default:
		return nil
	}
}

// notifyPipeline notifies the app about pipeline-relevant changes.
func (s *Server) notifyPipeline(reason string) {
	if s.onPipelineChange != nil {
		s.onPipelineChange(reason)
	}
}

// pluginAbsVirtual converts the stored plugin rectangle into absolute virtual-desktop coordinates.
func (s *Server) pluginAbsVirtual(c calib.Calib) (calib.Rect, error) {
	monitors, err := s.listMonitors()
	if err != nil {
		return calib.Rect{}, err
	}
	monitorIndex := c.MonitorIndex
	if monitorIndex <= 0 {
		monitorIndex = s.session.Monitor()
	}
	m, ok := monitor.GetMonitorByIndex(monitors, monitorIndex)
	if !ok {
		return calib.Rect{}, fmt.Errorf("monitor %d not found", monitorIndex)
	}
	pluginAbs := calib.Normalize(c.PluginAbs)
	pluginAbs.X += m.X
	pluginAbs.Y += m.Y
	return pluginAbs, nil
}

// chatRectAbsFromPlugin converts the relative chat rect to absolute coordinates using an absolute plugin rectangle.
func chatRectAbsFromPlugin(pluginAbs calib.Rect, chatRel calib.Rect) calib.Rect {
	pluginAbs = calib.Normalize(pluginAbs)
	chat := calib.Normalize(chatRel)
	return calib.Rect{
		X: pluginAbs.X + chat.X,
		Y: pluginAbs.Y + chat.Y,
		W: chat.W,
		H: chat.H,
	}
}

// buildPresetupActions returns basic actions for presetup mode.
func buildPresetupActions(kind string, absX, absY int) []Action {
	switch kind {
	case "down":
		return []Action{{Type: ActClick, X: absX, Y: absY}}
	case "move":
		return []Action{{Type: ActMove, X: absX, Y: absY}}
	default:
		return nil
	}
}
