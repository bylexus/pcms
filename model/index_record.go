package model

type IndexedPage struct {
	Route           string
	ParentPageRoute *string
	Title           string
	IndexFile       string
	MetadataJSON    string
}

type IndexedFile struct {
	Route           string
	ParentPageRoute string
	FileName        string
	MimeType        string
	FileSize        int64
	MetadataJSON    string
}

type IndexSnapshot struct {
	Pages []IndexedPage
	Files []IndexedFile
}
