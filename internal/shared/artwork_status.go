package shared

//go:generate go-enum --values --names --nocase

// ArtworkStatus
/*
ENUM(
cached
posting
posted
)
*/
type ArtworkStatus string
