package swagger

import "embed"

// swaggerDocs provides embedded access to generated swagger assets.
//
//go:embed docs/*
var swaggerDocs embed.FS
