package sqlstore_test

import (
	"testing"

	"github.com/hydrohq/hydrogo/internal/jobs/sqlstore"
	news "github.com/hydrohq/hydrogo/modules/news"
)

func TestQueueImplementsNewsDurableContract(t *testing.T) {
	var _ news.DurableQueue = sqlstore.Queue{}
}
