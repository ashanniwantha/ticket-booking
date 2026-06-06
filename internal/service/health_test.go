//go:build integration

package service_test

import (
	"context"
	"testing"

	"github.com/ashanniwantha/ticket-booking/internal/test"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConnection(t *testing.T) {
	ctx := context.Background()

	pool, err := test.NewTestPool(ctx)
	require.NoError(t, err, "failed to connect to test database")
	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(ctx), "test database not reachable")

	rdb, err := test.NewTestRedis(ctx)
	require.NoError(t, err, "failed to connect to test redis")
	t.Cleanup(func() { rdb.Close() })

	require.NoError(t, rdb.Ping(ctx).Err(), "test redis not reachable")
}
