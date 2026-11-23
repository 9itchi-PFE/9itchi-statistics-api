package repositories

// StatisticsRepository is the only layer that touches PostgreSQL for statistics data.
// Use store.Conn(ctx), a context from QueryTimeout, and return MapError(err).
type StatisticsRepository interface {
}

type statisticsRepository struct {
	store *Store
}

func NewStatisticsRepository(store *Store) StatisticsRepository {
	return &statisticsRepository{store: store}
}
