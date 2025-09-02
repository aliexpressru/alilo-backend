package ammo

type File struct {
	AmmoFile    []byte
	Name        string
	BucketName  string
	ContentType string
	FinishChan  chan bool
}
