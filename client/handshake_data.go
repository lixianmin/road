package client

/********************************************************************
created:    2020-09-03
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

// HandshakeClientData represents information about the client sent on the handshake.
type HandshakeClientData struct {
	Platform    string `json:"platform"`
	LibVersion  string `json:"libVersion"`
	BuildNumber string `json:"clientBuildNumber"`
	Version     string `json:"clientVersion"`
}

// HandshakeRequest represents information about the handshake sent by the client.
// `sys` corresponds to information independent from the app and `user` information
// that depends on the app and is customized by the user.
type HandshakeRequest struct {
	Sys  HandshakeClientData    `json:"sys"`
	User map[string]interface{} `json:"user,omitempty"`
}