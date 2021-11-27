package models

//Config - структура для конфигурации
type Config struct {
	MaxDepth       uint64
	MaxResults     int
	MaxErrors      int
	Url            string
	RequestTimeout int
	GlobalTimeout  int
}
