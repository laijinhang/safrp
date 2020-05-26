package common

type Config struct {
	IP           string
	ExtranetPort string
	ServerPort   string
	Protocol     string
	PipeNum      uint8
}

type DataPackage struct {
	Number int
	Data []byte
}