package model

type IndexedPage struct {
	Route           string
	ParentPageRoute *string
	Title           string
	IndexFile       string
	Metadata        map[string]any
}

type IndexedFile struct {
	Route           string
	ParentPageRoute string
	FileName        string
	MimeType        string
	FileSize        int64
}

type IndexSnapshot struct {
	Pages []IndexedPage
	Files []IndexedFile
}
