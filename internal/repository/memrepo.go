package repository

type Storage interface{
	GetData(string, string) (string, error)
	SetData(string, string, string) error
}

type MemRepo struct {
	storage Storage
}

func NewMemRepo(config Storage) *MemRepo {
	return &MemRepo{
		storage: config,
	}
}

func (mr *MemRepo) GetData(metricType, key string) (string, error){
	return mr.storage.GetData(metricType, key)
}

func (mr *MemRepo) SetData(metricType, key, value string) error{
	return mr.storage.SetData(metricType, key, value)
}