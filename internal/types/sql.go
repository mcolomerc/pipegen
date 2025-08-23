package types

// SQLStatement represents a FlinkSQL statement
type SQLStatement struct {
	Name     string
	Content  string
	FilePath string
	Order    int
}
