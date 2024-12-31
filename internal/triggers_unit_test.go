package pp

// import (
// 	"strings"
// 	"testing"
// )

// func TestFileFilter(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		context  RunContext
// 		testPath string
// 		want     bool
// 	}{
// 		{
// 			name: "exact match in watch list",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{"test.txt"},
// 							Ignore:         []string{},
// 							ContainFilters: []string{},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "test.txt",
// 			want:     true,
// 		},
// 		{
// 			name: "exact match in ignore list",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{},
// 							Ignore:         []string{"ignore.txt"},
// 							ContainFilters: []string{"*.txt"},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "ignore.txt",
// 			want:     false,
// 		},
// 		{
// 			name: "wildcard match with .test extension",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{},
// 							Ignore:         []string{},
// 							ContainFilters: []string{"*.test"},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "example.test",
// 			want:     true,
// 		},
// 		{
// 			name: "ignore pattern takes precedence over contain filter",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{},
// 							Ignore:         []string{"*.bak"},
// 							ContainFilters: []string{"*.test"},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "example.test.bak",
// 			want:     false,
// 		},
// 		{
// 			name: "watch list takes precedence over ignore list",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{"special.bak"},
// 							Ignore:         []string{"*.bak"},
// 							ContainFilters: []string{},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "special.bak",
// 			want:     true,
// 		},
// 		{
// 			name: "multiple wildcards in pattern",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{},
// 							Ignore:         []string{},
// 							ContainFilters: []string{"*test*.go"},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "mytest123.go",
// 			want:     true,
// 		},
// 		{
// 			name: "no matches should return false",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{},
// 							Ignore:         []string{},
// 							ContainFilters: []string{"*.test"},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "example.txt",
// 			want:     false,
// 		},
// 		{
// 			name: "case sensitive matching",
// 			context: RunContext{
// 				Process: Process{
// 					Trigger: Trigger{
// 						FileSystem: FileSystem{
// 							Watch:          []string{},
// 							Ignore:         []string{},
// 							ContainFilters: []string{"*.TEST"},
// 						},
// 					},
// 				},
// 			},
// 			testPath: "example.test",
// 			want:     false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			filter := tt.context.fileFilter()
// 			got := filter(tt.testPath)
// 			if got != tt.want {
// 				t.Errorf("fileFilter() = %v, want %v for path %v", got, tt.want, tt.testPath)
// 			}
// 		})
// 	}
// }

// // TestFileFilterWithSpecialCharacters tests handling of special regex characters in patterns
// func TestFileFilterWithSpecialCharacters(t *testing.T) {
// 	context := RunContext{
// 		Process: Process{
// 			Trigger: Trigger{
// 				FileSystem: FileSystem{
// 					ContainFilters: []string{"*.test+", "test[123].txt", "test.(txt)"},
// 				},
// 			},
// 		},
// 	}

// 	tests := []struct {
// 		name     string
// 		testPath string
// 		want     bool
// 	}{
// 		{"plus sign in pattern", "file.test+", true},
// 		{"square brackets in pattern", "test[123].txt", true},
// 		{"parentheses in pattern", "test.(txt)", true},
// 		{"should not match regex interpretation", "test1.txt", false},
// 		{"should not match regex interpretation", "testX.txt", false},
// 	}

// 	filter := context.fileFilter()
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := filter(tt.testPath)
// 			if got != tt.want {
// 				t.Errorf("fileFilter() = %v, want %v for path %v", got, tt.want, tt.testPath)
// 			}
// 		})
// 	}
// }

// // TestFileFilterEdgeCases tests edge cases and boundary conditions
// func TestFileFilterEdgeCases(t *testing.T) {
// 	context := RunContext{
// 		Process: Process{
// 			Trigger: Trigger{
// 				FileSystem: FileSystem{
// 					Watch:          []string{""},
// 					Ignore:         []string{""},
// 					ContainFilters: []string{"*"},
// 				},
// 			},
// 		},
// 	}

// 	tests := []struct {
// 		name     string
// 		testPath string
// 		want     bool
// 	}{
// 		{"empty path", "", true},
// 		{"single character", "a", true},
// 		{"just wildcard pattern", "*", true},
// 		{"very long filename", strings.Repeat("a", 1000) + ".test", true},
// 	}

// 	filter := context.fileFilter()
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := filter(tt.testPath)
// 			if got != tt.want {
// 				t.Errorf("fileFilter() = %v, want %v for path %v", got, tt.want, tt.testPath)
// 			}
// 		})
// 	}
// }
