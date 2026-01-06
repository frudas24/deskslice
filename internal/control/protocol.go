// Package control handles input protocol and gesture mapping.
package control

// Rect represents a rectangle sent by the client UI.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Message is a control websocket payload.
type Message struct {
	T       string  `json:"t"`
	ID      int     `json:"id,omitempty"`
	X       float64 `json:"x,omitempty"`
	Y       float64 `json:"y,omitempty"`
	WheelX  int     `json:"wheelX,omitempty"`
	WheelY  int     `json:"wheelY,omitempty"`
	Text    string  `json:"text,omitempty"`
	Mode    string  `json:"mode,omitempty"`
	Video   string  `json:"video,omitempty"`
	Idx     int     `json:"idx,omitempty"`
	Step    string  `json:"step,omitempty"`
	Rect    *Rect   `json:"rect,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}
