package definition

import (
	"errors"
	"time"
)

var BaseTTL = 3 * time.Minute
var SortedSetTTL = 1 * time.Minute
var ItemPerPage = int64(5)

var NotFound = errors.New("item not found")
