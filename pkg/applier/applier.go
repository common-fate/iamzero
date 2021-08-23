package applier

// ApplierOutput is the output from the iamzero-applier command
type ApplierOutput []ChangedFile

// ChangedFile is a file change made by the applier to a specific source code file.
type ChangedFile struct {
	// the path of the file which has been modified
	Path string `json:"path"`
	// the new contents of the file
	Contents string `json:"contents"`
}
