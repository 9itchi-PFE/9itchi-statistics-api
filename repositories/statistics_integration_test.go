//go:build integration

package repositories

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIncrementAtomicallyUpsertsByNameAndDay(t *testing.T) {
	repo := NewStatisticsRepository(newTestStore(t))
	ctx := context.Background()
	day := time.Date(2026, 1, 2, 22, 0, 0, 0, time.FixedZone("offset", 2*60*60))

	const increments = 12
	errs := make(chan error, increments)
	var wg sync.WaitGroup
	for i := 0; i < increments; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Increment(ctx, "email_sent", day)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	rows, err := repo.List(ctx, day.UTC(), day.UTC())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(increments), rows[0].Value)
}
