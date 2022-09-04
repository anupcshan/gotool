module github.com/anupcshan/gotool/cmd

go 1.19

replace github.com/anupcshan/gotool => ../

require (
	github.com/anupcshan/gotool v0.0.0-00010101000000-000000000000
	github.com/google/go-github/v47 v47.0.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

require (
	github.com/google/go-querystring v1.1.0 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	gopkg.in/freddierice/go-losetup.v1 v1.0.0-20170407175016-fc9adea44124 // indirect
)
