package models

type Permission string

const (
	READ   Permission = "read"
	WRITE  Permission = "write"
	DELETE Permission = "delete"
	ADMIN  Permission = "admin"
)
