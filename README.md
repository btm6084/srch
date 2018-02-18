Go-Based regular-expression search

# Installation
```
go get -u github.com/btm6084/srch
```

You can add your $GOPATH/bin to your $PATH to access it directly.
```
export PATH=$PATH:$GOPATH/bin
```

# Usage
```
srch [flags] <search term> [search directory]

eg.

srch -i "case (.+)[:]" .
```

# Implemented Flags

| Flag | Type | Description | Example
--- | --- | --- | ---
| `i` | Bool | Case insensitive search | -i
| `v` | Bool | Inverse Search. Returns all lines that *do not* match the search term | -v
| `l` | Bool | File Name Only | -l
| `follow` | Bool | Follow symlinks when building file search list. | -follow
| `A` | Int | Returns X lines AFTER the match | -A=5
| `B` | Int | Returns X line BEFORE the match | -B=2
| `ignore-dir` | String | Comma separated list of directories to ignore | -ignore-dir=vendor,bower,node_modules

# srchrc configuration

Certain configuration options can be made permanent by adding a configuration file at /home/$USER/.srchrc/config.json

Currentl only ignore-dir is supported.

Example:
```
{
	"ignore-dir": [
		"vendor",
		"node_modules"
	]
}
```

# Vendoring
https://github.com/kardianos/govendor is the vendoring of choice, as I'm not a fan of how the official `dep` handles development dependencies right now.