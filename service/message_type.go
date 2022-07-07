package service

// Protocols defines the name of this protocol,
// which is supposed to be used to the subprotocol of Websockt streams.
var Protocols = []string{"webtty"}

const (
	// UnknownInput Unknown message type, maybe sent by a bug
	UnknownInput = '0'
	// Input User input typically from a keyboard
	Input = '1'
	// Ping to the server
	Ping = '2'
	// ResizeTerminal Notify that the browser size has been changed
	ResizeTerminal = '3'
)

const (
	// UnknownOutput Unknown message type, maybe set by a bug
	UnknownOutput = '0'
	// Output Normal output to the terminal
	Output = '1'
	// Pong to the browser
	Pong = '2'
	// SetWindowTitle Set window title of the terminal
	SetWindowTitle = '3'
	// SetPreferences Set terminal preference
	SetPreferences = '4'
	// SetReconnect Make terminal to reconnect
	SetReconnect = '5'
)
