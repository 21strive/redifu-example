package definition

import (
	"errors"
	"time"
)

var BaseTTL = 3 * time.Hour
var SortedSetTTL = 1 * time.Hour
var ItemPerPage = int64(5)

var NotFound = errors.New("item not found")
