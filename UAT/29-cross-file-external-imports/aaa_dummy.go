package crossfile

// Dummy is a dummy type to ensure this file is loaded first
// but has DIFFERENT imports than the FileSystem interface file.
type Dummy struct {
	Name string
}

// Format formats the dummy name.
