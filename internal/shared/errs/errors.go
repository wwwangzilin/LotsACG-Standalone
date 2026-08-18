package errs

import (
	"errors"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/search"
	"github.com/wwwangzilin/LotsACG-Standalone/internal/infra/tagging"
	"gorm.io/gorm"
)

var (
	ErrRecordNotFound         = gorm.ErrRecordNotFound
	ErrArtworkAlreadyExist    = errors.New("artwork already exists")
	ErrArtworkDeleted         = errors.New("artwork has been deleted")
	ErrAliasAlreadyUsed       = errors.New("alias already used")
	ErrSearchEngineNotEnabled = search.ErrNotEnabled
	ErrTaggingNotEnabled      = tagging.ErrNotEnabled
)
