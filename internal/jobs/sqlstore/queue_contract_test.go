package sqlstore_test

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	news "github.com/ardaozyildirim/hydrogo/modules/news"
)

func TestQueueImplementsNewsDurableContract(t *testing.T) {
	var _ news.DurableQueue = sqlstore.Queue{}
}
