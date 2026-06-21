package entity

type LoginStatus int

const (
	LoginStatusWaiting LoginStatus = iota
	LoginStatusScanning
	LoginStatusExpired
	LoginStatusSuccess
)
