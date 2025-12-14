package html

import "embed"

//go:embed templates/* static/*
var Embedded embed.FS
