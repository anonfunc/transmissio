package blackhole

import "testing"

func Test_blackholePathToDownloadDir(t *testing.T) {
	type args struct {
		file            string
		basePath        string
		baseDownloadDir string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Normal",
			args: args{
				file:            "/hole/test.mp4",
				basePath:        "/hole",
				baseDownloadDir: "/download",
			},
			want: "/download",
		},
		{
			name: "With Category",
			args: args{
				file:            "/hole/Movie/test.mp4",
				basePath:        "/hole",
				baseDownloadDir: "/download",
			},
			want: "/download/Movie",
		},
		{
			name: "With Nested Category",
			args: args{
				file:            "/hole/Movie/Foo/bar.mp4",
				basePath:        "/hole",
				baseDownloadDir: "/download",
			},
			want: "/download/Movie/Foo",
		},
		{
			name: "Blackhole under Download (Issue #2)",
			args: args{
				file:            "/export/nas/Downloads/Torrents/test.torrent",
				basePath:        "/export/nas/Downloads/Torrents/",
				baseDownloadDir: "/export/nas/Downloads/",
			},
			want: "/export/nas/Downloads",
		},
	}
	for _, tt2 := range tests {
		tt := tt2
		t.Run(tt.name, func(t *testing.T) {
			if got := blackholePathToDownloadDir(tt.args.file, tt.args.basePath, tt.args.baseDownloadDir); got != tt.want {
				t.Errorf("blackholePathToDownloadDir() = %v, want %v", got, tt.want)
			}
		})
	}
}
