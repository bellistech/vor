package cscore

import "testing/fstest"

func testSheetFS() fstest.MapFS {
	return fstest.MapFS{
		"sheets/shell/bash.md": &fstest.MapFile{
			Data: []byte("# Bash\n\nBourne Again Shell.\n\n## Variables\n\n```bash\nNAME=\"world\"\necho $NAME\n```\n\n## Functions\n\n```bash\ngreet() { echo \"hi\"; }\n```\n\n## See Also\n\n- zsh, fish\n"),
		},
		"sheets/shell/zsh.md": &fstest.MapFile{
			Data: []byte("# Zsh\n\nZ Shell with advanced features.\n\n## Plugins\n\noh-my-zsh is popular.\n\n## See Also\n\n- bash\n"),
		},
		"sheets/networking/curl.md": &fstest.MapFile{
			Data: []byte("# curl\n\nTransfer data with URLs.\n\n## GET Request\n\n```bash\ncurl https://example.com\n```\n\n## POST Request\n\n```bash\ncurl -X POST -d 'data' https://example.com\n```\n\n## See Also\n\n- wget\n"),
		},
	}
}

func testDetailFS() fstest.MapFS {
	return fstest.MapFS{
		"detail/shell/bash.md": &fstest.MapFile{
			Data: []byte("# The Mathematics of Bash\n\nDetailed theory of shell processes.\n\n## Process Model\n\nBash forks subshells. A pipeline of 3 commands creates 2 * 3 = 6 processes.\n\n## Prerequisites\n\n- terminal\n- linux-basics\n\n## Complexity\n\nO(1) for variable lookup, O(n) for globbing.\n"),
		},
	}
}

// initTestRegistry sets up cscore with test data. Call in every test.
func initTestRegistry() {
	resetForTesting()
	if err := Init(testSheetFS(), testDetailFS()); err != nil {
		panic("initTestRegistry: " + err.Error())
	}
}
