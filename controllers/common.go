package controllers

type Instance interface {
	ClientName() string
	NameSpace() string
	StatusClientID() string
	Description() string
	SecretName() string
}
