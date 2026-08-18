package query

type PicturesPhash struct {
	Input    string
	Distance int
	Limit    int
}

type PicturesORB struct {
	Input      string
	MinMatches int
	MinScore   float64
	Limit      int
}
